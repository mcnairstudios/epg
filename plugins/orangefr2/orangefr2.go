package orangefr2

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

func (p *provider) Name() string        { return "chaines-tv.orange.fr" }
func (p *provider) Description() string { return "France Orange TV" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "chaines-tv.orange.fr")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour).UTC()
	end := start.Add(24 * time.Hour)

	url := fmt.Sprintf(
		"https://rp-ott-mediation-tv.woopic.com/api-gw/live/v3/applications/STB4PC/programs?groupBy=channel&includeEmptyChannels=false&period=%d,%d&after=%s&limit=1",
		start.UnixMilli(),
		end.UnixMilli(),
		channel.SiteID,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")

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
	var response map[string][]programItem
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil
	}

	items, ok := response[channel.SiteID]
	if !ok {
		return nil, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		start := time.Unix(item.DiffusionDate, 0).UTC()
		stop := start.Add(time.Duration(item.Duration) * time.Second)

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "chaines-tv.orange.fr",
			Title:       item.Title,
			Description: item.Synopsis,
			Category:    item.GenreDetailed,
			Image:       parseImage(item.Covers),
			Start:       start,
			Stop:        stop,
			Lang:        channel.Lang,
		}

		if item.Season != nil {
			p.Season = item.Season.Number
			if item.Season.Serie != nil {
				p.SubTitle = item.Season.Serie.Title
			}
		}

		if item.EpisodeNumber > 0 {
			p.Episode = item.EpisodeNumber
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func parseImage(covers []cover) string {
	if len(covers) == 0 {
		return ""
	}
	return covers[0].URL
}

type programItem struct {
	Title         string  `json:"title"`
	DiffusionDate int64   `json:"diffusionDate"`
	Duration      int64   `json:"duration"`
	Synopsis      string  `json:"synopsis"`
	GenreDetailed string  `json:"genreDetailed"`
	EpisodeNumber int     `json:"episodeNumber"`
	Season        *season `json:"season"`
	Covers        []cover `json:"covers"`
}

type season struct {
	Number int    `json:"number"`
	Serie  *serie `json:"serie"`
}

type serie struct {
	Title string `json:"title"`
}

type cover struct {
	Format string `json:"format"`
	URL    string `json:"url"`
}
