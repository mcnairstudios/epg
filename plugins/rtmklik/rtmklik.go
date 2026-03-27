package rtmklik

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "rtmklik.rtm.gov.my" }
func (p *provider) Description() string { return "RTM Klik Malaysia EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "rtmklik.rtm.gov.my")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	dateStr := date.Format("2006-01-02")
	url := fmt.Sprintf(
		"https://rtm.glueapi.io/v3/epg/%s/ChannelSchedule?dateStart=%s&dateEnd=%s&timezone=0",
		channel.SiteID, dateStr, dateStr,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return []domain.Program{}, nil
	}

	if len(resp.Schedule) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(resp.Schedule))
	for _, item := range resp.Schedule {
		start, err := time.Parse("2006-01-02T15:04:05", item.DateTimeStart)
		if err != nil {
			continue
		}

		stop, err := time.Parse("2006-01-02T15:04:05", item.DateTimeEnd)
		if err != nil {
			continue
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "rtmklik.rtm.gov.my",
			Title:       item.ProgramTitle,
			Description: item.Description,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		})
	}

	return programs, nil
}

type apiResponse struct {
	Schedule []scheduleItem `json:"schedule"`
}

type scheduleItem struct {
	ProgramTitle string `json:"programTitle"`
	Description  string `json:"description"`
	DateTimeStart string `json:"dateTimeStart"`
	DateTimeEnd   string `json:"dateTimeEnd"`
}
