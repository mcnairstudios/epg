package nhlcom

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

func (p *provider) Name() string        { return "nhl.com" }
func (p *provider) Description() string { return "NHL Network TV schedule" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "nhl.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://api-web.nhle.com/v1/network/tv-schedule/%s", date.Format("2006-01-02"))

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
		return nil, fmt.Errorf("HTTP %d from nhlcom", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	programs := make([]domain.Program, 0, len(response.Broadcasts))
	for _, item := range response.Broadcasts {
		p := domain.Program{
			Channel:  channel.XmltvID,
			Source:   "nhl.com",
			Title:    item.Title,
			Category: "Sports",
			Lang:     channel.Lang,
		}

		if item.Description != item.Title {
			p.Description = item.Description
		}

		if t, err := time.Parse("2006-01-02T15:04:05", item.StartTime); err == nil {
			p.Start = t.UTC()
		}
		if t, err := time.Parse("2006-01-02T15:04:05", item.EndTime); err == nil {
			p.Stop = t.UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Broadcasts []broadcast `json:"broadcasts"`
}

type broadcast struct {
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (p *provider) Timezone() string { return "America/New_York" }
