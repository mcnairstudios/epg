package clarotvmais

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

func (p *provider) Name() string        { return "clarotvmais.com.br" }
func (p *provider) Description() string { return "Claro TV+ Brazil" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "clarotvmais-1", Name: "Discovery Kids", Source: "clarotvmais.com.br", SiteID: "1", Lang: "pt", XmltvID: "DiscoveryKids.br"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://www.clarotvmais.com.br/avsclient/1.2/epg/livechannels?types=&channelIds=%s&startTime=%d&endTime=%d&location=SAO%%20PAULO,AMAZONAS&channel=PCTV",
		channel.SiteID, date.Unix(), date.AddDate(0, 0, 1).Unix(),
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

	if len(response.Response.LiveChannels) == 0 {
		return []domain.Program{}, nil
	}

	schedules := response.Response.LiveChannels[0].Schedules
	if len(schedules) <= 1 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(schedules))
	for _, s := range schedules {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "clarotvmais.com.br",
			Title:       s.Title,
			Description: s.Description,
			Lang:        channel.Lang,
			Start:       time.Unix(s.StartTime, 0).UTC(),
			Stop:        time.Unix(s.EndTime, 0).UTC(),
		}

		if s.SeasonNumber > 0 {
			p.Season = s.SeasonNumber
		}
		if s.EpisodeNumber > 0 {
			p.Episode = s.EpisodeNumber
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Response responseData `json:"response"`
}

type responseData struct {
	LiveChannels []liveChannel `json:"liveChannels"`
}

type liveChannel struct {
	Schedules []scheduleItem `json:"schedules"`
}

type scheduleItem struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	StartTime     int64  `json:"startTime"`
	EndTime       int64  `json:"endTime"`
	SeasonNumber  int    `json:"seasonNumber"`
	EpisodeNumber int    `json:"episodeNumber"`
}
