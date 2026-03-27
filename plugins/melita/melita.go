package melita

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

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

func (p *provider) Name() string        { return "melita.com" }
func (p *provider) Description() string { return "Melita Malta cable TV provider" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "melita.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour).UTC()
	end := start.Add(24 * time.Hour)

	url := fmt.Sprintf(
		"https://androme.melitacable.com/api/epg/v1/schedule/channel/%s/from/%s/until/%s",
		channel.SiteID,
		start.Format("2006-01-02T15:04-07:00"),
		end.Format("2006-01-02T15:04-07:00"),
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

	if response.Schedules == nil || response.Programs == nil || response.Seasons == nil || response.Series == nil {
		return []domain.Program{}, nil
	}

	programMap := make(map[string]programEntry)
	for _, p := range response.Programs {
		programMap[p.ID] = p
	}

	seasonMap := make(map[string]seasonEntry)
	for _, s := range response.Seasons {
		seasonMap[s.ID] = s
	}

	seriesMap := make(map[string]seriesEntry)
	for _, s := range response.Series {
		seriesMap[s.ID] = s
	}

	programs := make([]domain.Program, 0)
	for _, sched := range response.Schedules {
		prog, ok := programMap[sched.Program]
		if !ok {
			continue
		}
		season, ok := seasonMap[prog.Season]
		if !ok {
			continue
		}
		_, ok = seriesMap[season.Series]
		if !ok {
			continue
		}

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "melita.com",
			Title:       prog.Title,
			Description: prog.ShortSynopsis,
			Season:      season.Season,
			Episode:     prog.Episode,
			Lang:        channel.Lang,
		}

		if len(prog.Tags) > 0 {
			p.Category = prog.Tags[0]
		}

		if prog.PosterImage != "" {
			p.Image = prog.PosterImage + "?form=epg-card-6"
		}

		if sched.Published.Start != "" {
			if t, err := time.Parse(time.RFC3339, sched.Published.Start); err == nil {
				p.Start = t.UTC()
			}
		}

		if sched.Published.End != "" {
			if t, err := time.Parse(time.RFC3339, sched.Published.End); err == nil {
				p.Stop = t.UTC()
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Schedules []scheduleEntry `json:"schedules"`
	Programs  []programEntry  `json:"programs"`
	Seasons   []seasonEntry   `json:"seasons"`
	Series    []seriesEntry   `json:"series"`
}

type scheduleEntry struct {
	ID        string    `json:"id"`
	Published published `json:"published"`
	Program   string    `json:"program"`
}

type published struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type programEntry struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	ShortSynopsis string   `json:"shortSynopsis"`
	PosterImage   string   `json:"posterImage"`
	Episode       int      `json:"episode"`
	Season        string   `json:"season"`
	Tags          []string `json:"tags"`
}

type seasonEntry struct {
	ID     string `json:"id"`
	Season int    `json:"season"`
	Series string `json:"series"`
}

type seriesEntry struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}
