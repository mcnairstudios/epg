package toonami

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

func (p *provider) Name() string        { return "toonamiaftermath.com" }
func (p *provider) Description() string { return "Toonami Aftermath streaming" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "toonamiaftermath.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	// Step 1: Get playlists for this week
	nextDay := date.Add(24 * time.Hour)
	playlistsURL := fmt.Sprintf(
		"https://api.toonamiaftermath.com/playlists?scheduleName=%s&startDate=%s&thisWeek=true&weekStartDay=monday",
		channel.SiteID, nextDay.Format(time.RFC3339),
	)

	req, err := http.NewRequest(http.MethodGet, playlistsURL, nil)
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

	// Step 2: Find matching playlist by date
	var playlists []playlistEntry
	if err := json.Unmarshal(body, &playlists); err != nil {
		return nil, err
	}

	dateStr := date.Format("2006-01-02")
	var playlistID string
	for _, pl := range playlists {
		if len(pl.StartDate) >= 10 && pl.StartDate[:10] == dateStr {
			playlistID = pl.ID
			break
		}
	}

	if playlistID == "" {
		return []domain.Program{}, nil
	}

	// Step 3: Fetch the playlist by ID
	detailURL := fmt.Sprintf("https://api.toonamiaftermath.com/playlist?id=%s&addInfo=true", playlistID)

	req2, err := http.NewRequest(http.MethodGet, detailURL, nil)
	if err != nil {
		return nil, err
	}

	resp2, err := p.client.Do(req2)
	if err != nil {
		return nil, err
	}
	defer resp2.Body.Close()

	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body2, channel)
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

type playlistEntry struct {
	ID        string `json:"_id"`
	StartDate string `json:"startDate"`
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
