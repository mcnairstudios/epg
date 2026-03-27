package melita

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

func (p *provider) Name() string        { return "melita.com" }
func (p *provider) Description() string { return "Melita Cable Malta" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "melita-tvm", Name: "TVM", Source: "melita.com", SiteID: "4d40a9f9-12fd-4f03-8072-61c637ff6995", Lang: "en", XmltvID: "TVM.mt"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour).UTC()
	end := start.Add(24 * time.Hour)

	url := fmt.Sprintf(
		"https://androme.melitacable.com/api/epg/v1/schedule/channel/%s/from/%s/until/%s",
		channel.SiteID,
		start.Format("2006-01-02T15:04Z07:00"),
		end.Format("2006-01-02T15:04Z07:00"),
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
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return []domain.Program{}, nil
	}

	if resp.Schedules == nil || resp.Programs == nil || resp.Seasons == nil || resp.Series == nil {
		return []domain.Program{}, nil
	}

	programMap := make(map[string]programEntry)
	for _, p := range resp.Programs {
		programMap[p.ID] = p
	}
	seasonMap := make(map[string]seasonEntry)
	for _, s := range resp.Seasons {
		seasonMap[s.ID] = s
	}
	seriesMap := make(map[string]seriesEntry)
	for _, s := range resp.Series {
		seriesMap[s.ID] = s
	}

	var programs []domain.Program
	for _, sched := range resp.Schedules {
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
			Season:      season.SeasonNum,
			Episode:     prog.Episode,
			Lang:        channel.Lang,
		}

		if prog.PosterImage != "" {
			p.Image = prog.PosterImage + "?form=epg-card-6"
		}

		if len(prog.Tags) > 0 {
			p.Category = prog.Tags[0]
		}

		if sched.Published != nil {
			if sched.Published.Start != "" {
				if t, err := time.Parse(time.RFC3339, sched.Published.Start); err == nil {
					p.Start = t
				}
			}
			if sched.Published.End != "" {
				if t, err := time.Parse(time.RFC3339, sched.Published.End); err == nil {
					p.Stop = t
				}
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
	ID        string     `json:"id"`
	Published *published `json:"published"`
	Program   string     `json:"program"`
}

type published struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type programEntry struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	ShortSynopsis string  `json:"shortSynopsis"`
	PosterImage  string   `json:"posterImage"`
	Episode      int      `json:"episode"`
	Season       string   `json:"season"`
	Tags         []string `json:"tags"`
}

type seasonEntry struct {
	ID        string `json:"id"`
	SeasonNum int    `json:"season"`
	Series    string `json:"series"`
}

type seriesEntry struct {
	ID string `json:"id"`
}
