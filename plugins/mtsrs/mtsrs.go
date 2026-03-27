package mtsrs

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
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "mts.rs" }
func (p *provider) Description() string { return "MTS Serbia TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "mtsrs-rts_1_hd", Name: "RTS 1 HD", Source: "mts.rs", SiteID: "rts_1_hd", Lang: "sr", XmltvID: "RTS1HD.rs"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()
	url := fmt.Sprintf("https://mts.rs/hybris/ecommerce/b2c/v1/products/search?sort=pozicija-rastuce&searchQueryContext=CHANNEL_PROGRAM&query=:pozicija-rastuce:tip-kanala-radio:TV kanali:channelProgramDates:%s&pageSize=10000",
		d.Format("2006-01-02"))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return []domain.Program{}, nil
	}

	var channelData *product
	for i := range response.Products {
		if response.Products[i].Code == channel.SiteID {
			channelData = &response.Products[i]
			break
		}
	}

	if channelData == nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(channelData.Programs))
	for _, item := range channelData.Programs {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "mts.rs",
			Title:       item.Title,
			Category:    item.Category,
			Description: item.Description,
			Lang:        channel.Lang,
		}

		if item.Picture != nil {
			p.Image = item.Picture.URL
		}

		if item.Start != "" {
			if t, err := time.Parse("2006-01-02T15:04:05-07:00", item.Start); err == nil {
				p.Start = t.UTC()
			}
		}
		if item.End != "" {
			if t, err := time.Parse("2006-01-02T15:04:05-07:00", item.End); err == nil {
				p.Stop = t.UTC()
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Products []product `json:"products"`
}

type product struct {
	Code     string           `json:"code"`
	Name     string           `json:"name"`
	Programs []programEntry   `json:"programs"`
}

type programEntry struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Start       string   `json:"start"`
	End         string   `json:"end"`
	Picture     *picture `json:"picture"`
}

type picture struct {
	URL string `json:"url"`
}
