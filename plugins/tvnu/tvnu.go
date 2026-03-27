package tvnu

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

func (p *provider) Name() string        { return "tv.nu" }
func (p *provider) Description() string { return "Swedish TV schedule from tv.nu" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "tvnu-3sat", Name: "3sat", Source: "tv.nu", SiteID: "3sat", Lang: "sv", XmltvID: "3sat.de"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()

	url := fmt.Sprintf(
		"https://web-api.tv.nu/channels/%s/schedule?date=%s&fullDay=true",
		channel.SiteID,
		d.Format("2006-01-02"),
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
	var response APIResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	if response.Data == nil || response.Data.Broadcasts == nil {
		return nil, nil
	}

	programs := make([]domain.Program, 0, len(response.Data.Broadcasts))
	for _, item := range response.Data.Broadcasts {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tv.nu",
			Title:       item.Title,
			Description: item.Description,
			Season:      item.SeasonNumber,
			Episode:     item.EpisodeNumber,
			Lang:        channel.Lang,
		}

		if len(item.Genres) > 0 {
			names := make([]string, 0, len(item.Genres))
			for _, g := range item.Genres {
				names = append(names, g.Name)
			}
			p.Category = joinStrings(names)
		}

		if item.Broadcast != nil {
			if item.Broadcast.StartTime != 0 {
				p.Start = time.UnixMilli(item.Broadcast.StartTime).UTC()
			}
			if item.Broadcast.EndTime != 0 {
				p.Stop = time.UnixMilli(item.Broadcast.EndTime).UTC()
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func joinStrings(s []string) string {
	if len(s) == 0 {
		return ""
	}
	result := s[0]
	for i := 1; i < len(s); i++ {
		result += ", " + s[i]
	}
	return result
}

type APIResponse struct {
	Data *DataWrapper `json:"data"`
}

type DataWrapper struct {
	Broadcasts []BroadcastItem `json:"broadcasts"`
}

type BroadcastItem struct {
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	SeasonNumber  int        `json:"seasonNumber"`
	EpisodeNumber int        `json:"episodeNumber"`
	Genres        []Genre    `json:"genres"`
	Broadcast     *Broadcast `json:"broadcast"`
}

type Genre struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Broadcast struct {
	StartTime int64 `json:"startTime"`
	EndTime   int64 `json:"endTime"`
}
