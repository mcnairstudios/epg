package sportsnetca

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

func (p *provider) Name() string        { return "watch.sportsnet.ca" }
func (p *provider) Description() string { return "Sportsnet Canada" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "watch.sportsnet.ca")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://production-cdn.sportsnet.ca/api/schedules?channels=%s&date=%s&duration=24&hour=0",
		channel.SiteID, date.Format("2006-01-02"),
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
		return nil, fmt.Errorf("HTTP %d from sportsnetca", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response []channelSchedule
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	if len(response) == 0 || len(response[0].Schedules) == 0 {
		return []domain.Program{}, nil
	}

	schedules := response[0].Schedules
	programs := make([]domain.Program, 0, len(schedules))

	for _, s := range schedules {
		start, _ := time.Parse(time.RFC3339, s.StartDate)
		stop, _ := time.Parse(time.RFC3339, s.EndDate)

		if start.IsZero() {
			start, _ = time.Parse(time.RFC3339Nano, s.StartDate)
		}
		if stop.IsZero() {
			stop, _ = time.Parse(time.RFC3339Nano, s.EndDate)
		}

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "watch.sportsnet.ca",
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		}

		if s.Item != nil {
			p.Title = s.Item.Title
			p.Description = s.Item.ShortDescription
			if s.Item.Images != nil {
				p.Image = s.Item.Images.Tile
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type channelSchedule struct {
	ChannelID string     `json:"channelId"`
	Schedules []schedule `json:"schedules"`
}

type schedule struct {
	StartDate string      `json:"startDate"`
	EndDate   string      `json:"endDate"`
	Item      *scheduleItem `json:"item"`
}

type scheduleItem struct {
	Title            string  `json:"title"`
	ShortDescription string  `json:"shortDescription"`
	Images           *images `json:"images"`
}

type images struct {
	Tile      string `json:"tile"`
	Wallpaper string `json:"wallpaper"`
}

func (p *provider) Timezone() string { return "America/Toronto" }
