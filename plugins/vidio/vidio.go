package vidio

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

func (p *provider) Name() string        { return "vidio.com" }
func (p *provider) Description() string { return "Vidio.com Indonesian streaming service" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "vidio-204", Name: "SCTV", Source: "vidio.com", SiteID: "204", Lang: "id", XmltvID: "SCTV.id"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://api.vidio.com/livestreamings/%s/schedules?filter[date]=%s",
		channel.SiteID,
		date.Format("2006-01-02"),
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
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(response.Data))
	for _, item := range response.Data {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "vidio.com",
			Title:       item.Attributes.Title,
			Description: item.Attributes.Description,
			Image:       item.Attributes.ImageLandscapeURL,
			Lang:        channel.Lang,
		}

		if item.Attributes.StartTime != "" {
			if t, err := time.Parse(time.RFC3339, item.Attributes.StartTime); err == nil {
				p.Start = t.UTC()
			}
		}
		if item.Attributes.EndTime != "" {
			if t, err := time.Parse(time.RFC3339, item.Attributes.EndTime); err == nil {
				p.Stop = t.UTC()
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Data []scheduleItem `json:"data"`
}

type scheduleItem struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Attributes attributes `json:"attributes"`
}

type attributes struct {
	Title             string `json:"title"`
	StartTime         string `json:"start_time"`
	EndTime           string `json:"end_time"`
	Description       string `json:"description"`
	ImageLandscapeURL string `json:"image_landscape_url"`
}
