package frikanalen

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

func init() {
	plugin.Register(&provider{
		client: http.DefaultClient,
	})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "frikanalen.no" }
func (p *provider) Description() string { return "Frikanalen Norwegian community TV" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "frikanalen", Name: "Frikanalen", Source: "frikanalen.no", SiteID: "frikanalen", Lang: "no", XmltvID: "Frikanalen.no"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://frikanalen.no/api/scheduleitems/?date=%s&format=json&limit=100", date.Format("2006-01-02"))

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
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	programs := make([]domain.Program, 0, len(response.Results))
	for _, item := range response.Results {
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "frikanalen.no",
			Title:   item.Video.Name,
			Lang:    channel.Lang,
		}

		if item.Video.Header != "" {
			p.Description = item.Video.Header
		}

		if len(item.Video.Categories) > 0 {
			p.Category = item.Video.Categories[0]
		}

		if t, err := time.Parse(time.RFC3339, item.StartTime); err == nil {
			p.Start = t.UTC()
		}
		if t, err := time.Parse(time.RFC3339, item.EndTime); err == nil {
			p.Stop = t.UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Results []scheduleItem `json:"results"`
}

type scheduleItem struct {
	Video    video  `json:"video"`
	StartTime string `json:"starttime"`
	EndTime   string `json:"endtime"`
}

type video struct {
	Name       string   `json:"name"`
	Header     string   `json:"header"`
	Categories []string `json:"categories"`
}
