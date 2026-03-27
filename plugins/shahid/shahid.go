package shahid

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func (p *provider) Name() string        { return "shahid.mbc.net" }
func (p *provider) Description() string { return "Shahid MBC EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "shahid-996520", Name: "Al Aan TV", Source: "shahid.mbc.net", SiteID: "996520", Lang: "en", XmltvID: "AlAanTV.ae"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	dateStr := date.Format("2006-01-02")
	apiURL := fmt.Sprintf(
		"https://api2.shahid.net/proxy/v2.1/shahid-epg-api/?csvChannelIds=%s&from=%sT00:00:00.000Z&to=%sT23:59:59.999Z&country=SA&language=%s&Accept-Language=%s",
		channel.SiteID, dateStr, dateStr, channel.Lang, channel.Lang,
	)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
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
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return []domain.Program{}, nil
	}

	var items []epgItem
	for _, schedule := range resp.Items {
		if schedule.ChannelID == channel.SiteID {
			items = append(items, schedule.Items...)
		}
	}

	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		start, err := time.Parse(time.RFC3339Nano, item.ActualFrom)
		if err != nil {
			start, err = time.Parse("2006-01-02T15:04:05.000+00:00", item.ActualFrom)
			if err != nil {
				continue
			}
		}

		stop, err := time.Parse(time.RFC3339Nano, item.ActualTo)
		if err != nil {
			stop, err = time.Parse("2006-01-02T15:04:05.000+00:00", item.ActualTo)
			if err != nil {
				continue
			}
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "shahid.mbc.net",
			Title:       item.Title,
			Description: item.Description,
			Image:       removeParams(item.ProductPoster),
			Season:      item.SeasonNumber,
			Episode:     item.EpisodeNumber,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		})
	}

	return programs, nil
}

func removeParams(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

type apiResponse struct {
	Items []channelSchedule `json:"items"`
}

type channelSchedule struct {
	ChannelID string    `json:"channelId"`
	Items     []epgItem `json:"items"`
}

type epgItem struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	ActualFrom    string `json:"actualFrom"`
	ActualTo      string `json:"actualTo"`
	ProductPoster string `json:"productPoster"`
	SeasonNumber  int    `json:"seasonNumber,string"`
	EpisodeNumber int    `json:"episodeNumber"`
}
