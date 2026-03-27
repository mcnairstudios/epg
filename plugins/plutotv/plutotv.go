package plutotv

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

func (p *provider) Name() string        { return "pluto.tv" }
func (p *provider) Description() string { return "Pluto TV free streaming service" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "pluto-5ee92e72fb286e0007285fec", Name: "Naruto", Source: "pluto.tv", SiteID: "5ee92e72fb286e0007285fec", Lang: "es", Genre: "Anime", XmltvID: "Naruto.pluto.tv"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour).UTC()
	end := start.Add(72 * time.Hour)

	url := fmt.Sprintf(
		"https://api.pluto.tv/v2/channels/%s?start=%s&stop=%s",
		channel.SiteID,
		start.Format(time.RFC3339Nano),
		end.Format(time.RFC3339Nano),
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

	programs := make([]domain.Program, 0, len(response.Timelines))
	for _, item := range response.Timelines {
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "pluto.tv",
			Title:   item.Title,
			Lang:    channel.Lang,
		}

		if item.Start != "" {
			if t, err := time.Parse(time.RFC3339Nano, item.Start); err == nil {
				p.Start = t
			}
		}
		if item.Stop != "" {
			if t, err := time.Parse(time.RFC3339Nano, item.Stop); err == nil {
				p.Stop = t
			}
		}

		if item.Episode != nil {
			p.SubTitle = item.Episode.Name
			p.Description = item.Episode.Description
			p.Episode = item.Episode.Number
			p.Season = item.Episode.Season

			if item.Episode.Genre != "" {
				p.Category = item.Episode.Genre
			}
			if item.Episode.Series != nil && item.Episode.Series.Tile != nil {
				p.Image = item.Episode.Series.Tile.Path
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Timelines []timeline `json:"timelines"`
}

type timeline struct {
	Start   string   `json:"start"`
	Stop    string   `json:"stop"`
	Title   string   `json:"title"`
	Episode *episode `json:"episode"`
}

type episode struct {
	Number      int    `json:"number"`
	Season      int    `json:"season"`
	Description string `json:"description"`
	Genre       string `json:"genre"`
	SubGenre    string `json:"subGenre"`
	Rating      string `json:"rating"`
	Name        string `json:"name"`
	Clip        *clip  `json:"clip"`
	Series      *series `json:"series"`
}

type clip struct {
	Actors             []string `json:"actors"`
	Directors          []string `json:"directors"`
	OriginalReleaseDate string  `json:"originalReleaseDate"`
}

type series struct {
	Name string `json:"name"`
	Tile *tile  `json:"tile"`
}

type tile struct {
	Path string `json:"path"`
}
