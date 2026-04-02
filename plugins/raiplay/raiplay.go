package raiplay

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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
	plugin.Register(&provider{
		client: httpclient.Browser(),
	})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "raiplay.it" }
func (p *provider) Description() string { return "Rai Play Italian TV schedule" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "raiplay.it")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()

	url := fmt.Sprintf(
		"https://www.raiplay.it/palinsesto/app/%s/%s.json",
		channel.SiteID,
		d.Format("02-01-2006"),
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from raiplay", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

var romeLocation = mustLoadLocation("Europe/Rome")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	var response ScheduleResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil
	}

	if response.Events == nil {
		return nil, nil
	}

	programs := make([]domain.Program, 0, len(response.Events))
	dateStr := date.Format("2006-01-02")

	for _, item := range response.Events {
		if item.Name == "" || item.Hour == "" || item.DurationInMinutes == "" {
			continue
		}

		start, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+item.Hour, romeLocation)
		if err != nil {
			continue
		}

		duration := parseDurationMinutes(item.DurationInMinutes)
		stop := start.Add(time.Duration(duration) * time.Minute)

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "raiplay.it",
			Title:       item.Name,
			Description: item.Description,
			SubTitle:    item.EpisodeTitle,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		}

		p.Season = parseBoundedInt(item.Season, 2)
		p.Episode = parseBoundedInt(item.Episode, 3)

		if item.Image != "" {
			p.Image = "https://www.raiplay.it" + item.Image
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func parseDurationMinutes(s string) int {
	for i, c := range s {
		if c < '0' || c > '9' {
			s = s[:i]
			break
		}
	}
	n, _ := strconv.Atoi(s)
	return n
}

func parseBoundedInt(s string, maxDigits int) int {
	if s == "" || len(s) > maxDigits {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

type ScheduleResponse struct {
	Events []Event `json:"events"`
}

type Event struct {
	Name              string  `json:"name"`
	EpisodeTitle      string  `json:"episode_title"`
	Episode           string  `json:"episode"`
	Season            string  `json:"season"`
	Description       string  `json:"description"`
	Hour              string  `json:"hour"`
	DurationInMinutes string  `json:"duration_in_minutes"`
	Image             string  `json:"image"`
	Weblink           string  `json:"weblink"`
	EventWeblink      string  `json:"event_weblink"`
	Program           *RaiProgram `json:"program"`
}

type RaiProgram struct {
	Name string `json:"name"`
}

func (p *provider) Timezone() string { return "Europe/Rome" }
