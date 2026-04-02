package epgmaster

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

const token = "1610283054"

func init() {
	plugin.Register(&provider{
		client: httpclient.Browser(),
	})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "epgmaster.com" }
func (p *provider) Description() string { return "EPG Master TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "epgmaster.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://epgmaster.com/api/channels/%s/epgs?token=%s", channel.SiteID, token)

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
		return nil, fmt.Errorf("HTTP %d from epgmaster", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	var groups []dateGroup
	if err := json.Unmarshal(data, &groups); err != nil {
		return []domain.Program{}, nil
	}

	dateStr := date.Format("2006-01-02")
	var items []epgToken
	for _, g := range groups {
		if g.Date == dateStr {
			items = g.EPGTokenList
			break
		}
	}

	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		start := parseDateTime(item.StartDate, item.StartTime)
		stop := parseDateTime(item.StartDate, item.EndTime)

		if start.IsZero() || stop.IsZero() {
			continue
		}

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "epgmaster.com",
			Title:   item.ProgramName,
			Lang:    channel.Lang,
			Start:   start,
			Stop:    stop,
		})
	}

	return programs, nil
}

func parseDateTime(dateStr, timeStr string) time.Time {
	s := dateStr + " " + timeStr
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

type dateGroup struct {
	Date         string     `json:"date"`
	EPGTokenList []epgToken `json:"epgTokenList"`
}

type epgToken struct {
	ProgramName string `json:"programName"`
	StartDate   string `json:"startDate"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
}

func (p *provider) Timezone() string { return "Asia/Kathmandu" }
