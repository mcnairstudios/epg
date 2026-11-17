package allente

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
	"github.com/gavinmcnair/epg/internal/httpcache"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{client: httpcache.Wrap(httpclient.Browser())})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "allente.no" }
func (p *provider) Description() string { return "Allente Nordic TV provider" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "allente.no")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	country := strings.Split(channel.SiteID, "#")[0]
	url := fmt.Sprintf("https://cs-vcb.allente.%s/epg/events?date=%s", country, date.Format("2006-01-02"))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from allente", resp.StatusCode)
	}

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

	channelID := channel.SiteID
	if parts := strings.Split(channel.SiteID, "#"); len(parts) > 1 {
		channelID = parts[1]
	}

	var events []event
	for _, ch := range response.Channels {
		if ch.ID == channelID {
			events = ch.Events
			break
		}
	}

	programs := make([]domain.Program, 0, len(events))
	for _, item := range events {
		if item.Details == nil {
			continue
		}

		start, err := time.Parse(time.RFC3339, item.Time)
		if err != nil {
			continue
		}
		stop := start.Add(time.Duration(item.Details.Duration) * time.Minute)

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "allente.no",
			Title:       item.Title,
			Description: item.Details.Description,
			Image:       item.Details.Image,
			Season:      item.Details.Season,
			Episode:     item.Details.Episode,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		}

		if len(item.Details.Categories) > 0 {
			p.Category = item.Details.Categories[0]
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Channels []channelData `json:"channels"`
}

type channelData struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Events []event `json:"events"`
}

type event struct {
	ID      string       `json:"id"`
	Time    string       `json:"time"`
	Title   string       `json:"title"`
	Details *eventDetail `json:"details"`
}

type eventDetail struct {
	Title       string   `json:"title"`
	Image       string   `json:"image"`
	Description string   `json:"description"`
	Season      int      `json:"season"`
	Episode     int      `json:"episode"`
	Categories  []string `json:"categories"`
	Duration    int      `json:"duration,string"`
}

func (p *provider) Timezone() string { return "Europe/Oslo" }
