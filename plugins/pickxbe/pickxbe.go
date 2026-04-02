package pickxbe

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{client: httpclient.Browser()})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "pickx.be" }
func (p *provider) Description() string { return "Proximus Pickx Belgium EPG" }

const posterBaseURL = "https://experience-cache.cdi.streaming.proximustv.be/posterserver/poster/EPG/"

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "pickx.be")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://px-epg.azureedge.net/airings/%s/channel/%s?timezone=Europe%%2FBrussels",
		date.Format("2006-01-02"),
		channel.SiteID,
	)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Origin", "https://www.pickx.be")
	req.Header.Set("Referer", "https://www.pickx.be/")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from pickxbe", resp.StatusCode)
	}

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

	var items []airing
	if err := json.Unmarshal(data, &items); err != nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		start, _ := time.Parse(time.RFC3339Nano, item.ScheduleStart)
		stop, _ := time.Parse(time.RFC3339Nano, item.ScheduleEnd)

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "pickx.be",
			Title:       item.Program.Title,
			SubTitle:    item.Program.EpisodeTitle,
			Description: item.Program.Description,
			Season:      item.Program.SeasonNumber,
			Episode:     item.Program.EpisodeNumber,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		}

		if item.Program.TranslatedCategory != nil && channel.Lang != "" {
			if cat, ok := item.Program.TranslatedCategory[channel.Lang]; ok && cat != "" {
				p.Category = cat
			} else {
				p.Category = parseCategory(item.Program.Category)
			}
		} else {
			p.Category = parseCategory(item.Program.Category)
		}

		if item.Program.PosterFileName != "" {
			p.Image = posterBaseURL + item.Program.PosterFileName
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func parseCategory(cat string) string {
	parts := strings.Split(cat, ".")
	if len(parts) > 1 {
		return parts[1]
	}
	return cat
}

type airing struct {
	ScheduleStart string  `json:"programScheduleStart"`
	ScheduleEnd   string  `json:"programScheduleEnd"`
	Program       program `json:"program"`
}

type program struct {
	Title               string            `json:"title"`
	EpisodeTitle        string            `json:"episodeTitle"`
	Description         string            `json:"description"`
	Category            string            `json:"category"`
	PosterFileName      string            `json:"posterFileName"`
	SeasonNumber        int               `json:"seasonNumber"`
	EpisodeNumber       int               `json:"episodeNumber"`
	TranslatedCategory  map[string]string `json:"translatedCategory"`
}

func (p *provider) Timezone() string { return "Europe/Brussels" }
