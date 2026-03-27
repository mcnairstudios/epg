package maxtvgo

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

func (p *provider) Name() string        { return "maxtvgo.mk" }
func (p *provider) Description() string { return "MaxTV Go North Macedonia" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "maxtvgo-105", Name: "MaxTV", Source: "maxtvgo.mk", SiteID: "105", Lang: "mk", XmltvID: "MaxTV.mk"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://prd-static-mkt.spectar.tv/rev-1636968171/client_api.php/epg/list/instance_id/1/language/mk/channel_id/%s/start/%s/stop/%s/include_current/true/format/json",
		channel.SiteID,
		date.Format("20060102150405"),
		date.AddDate(0, 0, 1).Format("20060102150405"),
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

	programs := make([]domain.Program, 0, len(response.Programme))
	for _, item := range response.Programme {
		p := domain.Program{
			Channel:  channel.XmltvID,
			Source:   "maxtvgo.mk",
			Title:    item.Title,
			Category: item.Category,
			Lang:     channel.Lang,
		}

		if item.Desc != "" {
			p.Description = item.Desc
		}

		attrs := item.Attributes
		if attrs.Start != "" {
			if t, err := time.Parse("20060102150405 -0700", attrs.Start); err == nil {
				p.Start = t.UTC()
			}
		}
		if attrs.Stop != "" {
			if t, err := time.Parse("20060102150405 -0700", attrs.Stop); err == nil {
				p.Stop = t.UTC()
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Programme []programme `json:"programme"`
}

type programme struct {
	Attributes programmeAttrs `json:"@attributes"`
	Title      string         `json:"title"`
	Category   string         `json:"category"`
	Desc       string         `json:"desc"`
}

type programmeAttrs struct {
	Start string `json:"start"`
	Stop  string `json:"stop"`
}
