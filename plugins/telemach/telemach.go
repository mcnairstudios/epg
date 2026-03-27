package telemach

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

func (p *provider) Name() string        { return "epg.telemach.ba" }
func (p *provider) Description() string { return "Telemach Bosnia EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "telemach-1607", Name: "DW", Source: "epg.telemach.ba", SiteID: "1607", Lang: "bs", XmltvID: "DW.ba"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://api-web.ug-be.cdn.united.cloud/v1/public/events/epg?fromTime=%s&toTime=%s&communityId=12&languageId=59&cid=%s",
		date.Format("2006-01-02T15:04:05-00:00"),
		date.AddDate(0, 0, 1).Format("2006-01-02T15:04:05-00:00"),
		channel.SiteID,
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
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	programs := make([]domain.Program, 0)
	for _, rawItems := range raw {
		var items []epgItem
		if err := json.Unmarshal(rawItems, &items); err != nil {
			continue
		}

		for _, item := range items {
			p := domain.Program{
				Channel:     channel.XmltvID,
				Source:      "epg.telemach.ba",
				Title:       item.Title,
				Description: item.ShortDescription,
				Lang:        channel.Lang,
			}

			if item.SeasonNumber > 0 {
				p.Season = item.SeasonNumber
			}
			if item.EpisodeNumber > 0 {
				p.Episode = item.EpisodeNumber
			}

			if t, err := time.Parse(time.RFC3339Nano, item.StartTime); err == nil {
				p.Start = t.UTC()
			}
			if t, err := time.Parse(time.RFC3339Nano, item.EndTime); err == nil {
				p.Stop = t.UTC()
			}

			programs = append(programs, p)
		}
	}

	return programs, nil
}

type epgItem struct {
	Title            string `json:"title"`
	ShortDescription string `json:"shortDescription"`
	StartTime        string `json:"startTime"`
	EndTime          string `json:"endTime"`
	SeasonNumber     int    `json:"seasonNumber"`
	EpisodeNumber    int    `json:"episodeNumber"`
}
