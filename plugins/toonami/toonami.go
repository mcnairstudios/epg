package toonami

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

func (p *provider) Name() string        { return "toonamiaftermath.com" }
func (p *provider) Description() string { return "Toonami Aftermath streaming" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "toonami-est", Name: "Toonami Aftermath", Source: "toonamiaftermath.com", SiteID: "EST", Lang: "en", XmltvID: "ToonamiAftermath.us"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://api.toonamiaftermath.com/playlist?scheduleName=%s&date=%s", channel.SiteID, date.Format("2006-01-02"))

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

	if response.Playlist == nil {
		return []domain.Program{}, nil
	}

	var items []mediaItem
	for _, block := range response.Playlist.Blocks {
		items = append(items, block.MediaList...)
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "toonamiaftermath.com",
			Title:   item.Name,
			Lang:    channel.Lang,
		}

		if item.Info != nil {
			p.SubTitle = item.Info.Episode
			p.Image = item.Info.Image
		}

		if t, err := time.Parse(time.RFC3339Nano, item.StartDate); err == nil {
			p.Start = t.UTC()
		}
		if t, err := time.Parse(time.RFC3339Nano, item.EndDate); err == nil {
			p.Stop = t.UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Playlist *playlist `json:"playlist"`
}

type playlist struct {
	Blocks []block `json:"blocks"`
}

type block struct {
	MediaList []mediaItem `json:"mediaList"`
}

type mediaItem struct {
	Name      string     `json:"name"`
	StartDate string     `json:"startDate"`
	EndDate   string     `json:"endDate"`
	Info      *mediaInfo `json:"info"`
}

type mediaInfo struct {
	Episode string `json:"episode"`
	Image   string `json:"image"`
}
