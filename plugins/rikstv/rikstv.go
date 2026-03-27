package rikstv

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

func (p *provider) Name() string        { return "rikstv.no" }
func (p *provider) Description() string { return "RiksTV Norwegian digital TV" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "rikstv.no")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://play.rikstv.no/api/content-search/1/channel/%s/epg/%s",
		channel.SiteID,
		date.Format("2006-01-02"),
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
	var items []scheduleItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "rikstv.no",
			Title:       item.SeriesName,
			SubTitle:    item.Name,
			Description: item.Description,
			Image:       item.ImagePackURI,
			Lang:        channel.Lang,
		}

		if item.Season > 0 {
			p.Season = item.Season
		}
		if item.Episode > 0 {
			p.Episode = item.Episode
		}

		if len(item.Genres) > 0 {
			p.Category = item.Genres[0]
		}

		if t, err := time.Parse(time.RFC3339, item.BroadcastedTime); err == nil {
			p.Start = t.UTC()
		}
		if t, err := time.Parse(time.RFC3339, item.BroadcastedTimeEnd); err == nil {
			p.Stop = t.UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type scheduleItem struct {
	SeriesName        string   `json:"seriesName"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Season            int      `json:"season"`
	Episode           int      `json:"episode"`
	Genres            []string `json:"genres"`
	ImagePackURI      string   `json:"imagePackUri"`
	BroadcastedTime   string   `json:"broadcastedTime"`
	BroadcastedTimeEnd string  `json:"broadcastedTimeEnd"`
}
