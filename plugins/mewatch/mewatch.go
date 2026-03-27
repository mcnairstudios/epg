package mewatch

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

func (p *provider) Name() string        { return "mewatch.sg" }
func (p *provider) Description() string { return "mewatch Singapore streaming" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "mewatch-ch5", Name: "Channel 5", Source: "mewatch.sg", SiteID: "97098", Lang: "en", XmltvID: "Channel5.sg"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://cdn.mewatch.sg/api/schedules?channels=%s&date=%s&duration=24&ff=idp,ldp,rpt,cd&hour=00&intersect=true&lang=en&segments=all",
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var channels []channelSchedule
	if err := json.Unmarshal(data, &channels); err != nil {
		return nil, err
	}

	var schedules []schedule
	for _, ch := range channels {
		if ch.ChannelID == channel.SiteID {
			schedules = ch.Schedules
			break
		}
	}

	programs := make([]domain.Program, 0, len(schedules))
	for _, s := range schedules {
		if s.Item == nil {
			continue
		}

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "mewatch.sg",
			Title:       s.Item.Title,
			Description: s.Item.Description,
			Lang:        channel.Lang,
		}

		if s.Item.SeasonNumber > 0 {
			p.Season = s.Item.SeasonNumber
		}
		if s.Item.EpisodeNumber > 0 {
			p.Episode = s.Item.EpisodeNumber
		}
		if s.Item.Images.Tile != "" {
			p.Image = s.Item.Images.Tile
		}

		if t, err := time.Parse(time.RFC3339, s.StartDate); err == nil {
			p.Start = t.UTC()
		}
		if t, err := time.Parse(time.RFC3339, s.EndDate); err == nil {
			p.Stop = t.UTC()
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
	Title         string `json:"title"`
	Description   string `json:"description"`
	EpisodeNumber int    `json:"episodeNumber"`
	SeasonNumber  int    `json:"seasonNumber"`
	Images        images `json:"images"`
}

type images struct {
	Tile string `json:"tile"`
}
