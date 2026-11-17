package singtel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpcache"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{
		client: httpcache.Wrap(httpclient.Browser()),
	})
}

var singapore = time.FixedZone("Asia/Singapore", 8*60*60)

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "singtel.com" }
func (p *provider) Description() string { return "Singtel TV electronic programme guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "singtel.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	sgDate := date.In(singapore)
	url := fmt.Sprintf(
		"https://www.singtel.com/etc/singtel/public/tv/epg-parsed-data/%s.json",
		sgDate.Format("02012006"),
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from singtel", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response map[string][]scheduleItem
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	items, ok := response[channel.SiteID]
	if !ok {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		start, err := time.ParseInLocation("2006-01-02T15:04:05", item.StartDateTime, singapore)
		if err != nil {
			continue
		}

		duration, err := strconv.Atoi(item.Duration)
		if err != nil {
			continue
		}

		stop := start.Add(time.Duration(duration) * time.Second)

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "singtel.com",
			Title:       item.Program.Title,
			Description: item.Program.Description,
			Category:    item.Program.SubCategory,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

type scheduleItem struct {
	ChannelID     string  `json:"channelid"`
	Duration      string  `json:"duration"`
	ProgramID     string  `json:"programId"`
	Program       program `json:"program"`
	StartDateTime string  `json:"startDateTime"`
}

type program struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	SubCategory string `json:"subCategory"`
}

func (p *provider) Timezone() string { return "Asia/Singapore" }
