package tccuy

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

func (p *provider) Name() string        { return "programacion.tcc.com.uy" }
func (p *provider) Description() string { return "TCC Uruguay (149 channels)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "programacion.tcc.com.uy")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.UTC().Format("2006-01-02T15:04:05Z")
	end := date.Add(24 * time.Hour).UTC().Format("2006-01-02T15:04:05Z")

	url := fmt.Sprintf(
		"https://www.tccvivo.com.uy/api/v1/navigation_filter/1575/filter/?cable_operator=1&emission_start=%s&emission_end=%s&format=json",
		start, end,
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

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return []domain.Program{}, nil
	}

	if response.Results == nil {
		return []domain.Program{}, nil
	}

	var channelData *channelResult
	for i := range response.Results {
		if fmt.Sprintf("%d", response.Results[i].ID) == channel.SiteID {
			channelData = &response.Results[i]
			break
		}
	}

	if channelData == nil || len(channelData.Events) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(channelData.Events))
	for _, event := range channelData.Events {
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "programacion.tcc.com.uy",
			Title:   parseTitle(event),
			Lang:    channel.Lang,
		}

		p.Description = parseDescription(event)
		p.Category = parseCategories(event)
		p.Image = parseImage(event)
		p.Season = intOrZero(event.SeasonNumber)
		p.Episode = intOrZero(event.EpisodeNumber)

		if t, err := time.Parse(time.RFC3339Nano, event.EmissionStart); err == nil {
			p.Start = t.UTC()
		} else if t, err := time.Parse("2006-01-02T15:04:05Z", event.EmissionStart); err == nil {
			p.Start = t.UTC()
		}

		if t, err := time.Parse(time.RFC3339Nano, event.EmissionEnd); err == nil {
			p.Stop = t.UTC()
		} else if t, err := time.Parse("2006-01-02T15:04:05Z", event.EmissionEnd); err == nil {
			p.Stop = t.UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func parseTitle(event eventItem) string {
	for _, l := range event.Localized {
		if l.Language == "es" && l.Title != "" {
			return l.Title
		}
	}
	return event.OriginalTitle
}

func parseDescription(event eventItem) string {
	for _, l := range event.Localized {
		if l.Language == "es" && l.Description != "" {
			return l.Description
		}
	}
	return ""
}

func parseCategories(event eventItem) string {
	for _, g := range event.Genres {
		for _, l := range g.Localized {
			if l.Language == "es" && l.Name != "" {
				return l.Name
			}
		}
	}
	return ""
}

func parseImage(event eventItem) string {
	if len(event.Images) > 0 && event.Images[0].ImageMedia.File != "" {
		return "https:" + event.Images[0].ImageMedia.File
	}
	return ""
}

func intOrZero(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

type apiResponse struct {
	Results []channelResult `json:"results"`
}

type channelResult struct {
	ID     int         `json:"id"`
	Name   string      `json:"name"`
	Events []eventItem `json:"events"`
}

type eventItem struct {
	OriginalTitle string        `json:"original_title"`
	Localized     []localized   `json:"localized"`
	Genres        []genre       `json:"genres"`
	Images        []imageEntry  `json:"images"`
	Year          interface{}   `json:"year"`
	SeasonNumber  *int          `json:"season_number"`
	EpisodeNumber *int          `json:"episode_number"`
	EmissionStart string        `json:"emission_start"`
	EmissionEnd   string        `json:"emission_end"`
}

type localized struct {
	Language    string `json:"language"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Name        string `json:"name"`
}

type genre struct {
	Localized []localized `json:"localized"`
}

type imageEntry struct {
	ImageMedia imageMedia `json:"image_media"`
}

type imageMedia struct {
	File string `json:"file"`
}
