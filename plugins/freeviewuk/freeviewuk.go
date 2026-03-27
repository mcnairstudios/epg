package freeviewuk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "freeview.co.uk" }
func (p *provider) Description() string { return "Freeview UK TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "freeview.co.uk")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	networkID := strings.SplitN(channel.SiteID, "#", 2)[0]
	start := date.Truncate(24 * time.Hour).UTC()
	url := fmt.Sprintf(
		"https://www.freeview.co.uk/api/tv-guide?nid=%s&start=%d",
		networkID,
		start.Unix(),
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
		return nil, nil
	}

	channelID := ""
	parts := strings.SplitN(channel.SiteID, "#", 2)
	if len(parts) == 2 {
		channelID = parts[1]
	}

	var events []event
	for _, p := range response.Data.Programs {
		if p.ServiceID == channelID {
			events = p.Events
			break
		}
	}

	programs := make([]domain.Program, 0, len(events))
	for _, e := range events {
		prog := domain.Program{
			Channel:  channel.XmltvID,
			Source:   "freeview.co.uk",
			Title:    e.MainTitle,
			SubTitle: e.SecondaryTitle,
			Lang:     channel.Lang,
		}

		if e.ImageURL != "" {
			prog.Image = e.ImageURL + "?w=800"
		}

		if e.StartTime != "" {
			if t, err := time.Parse("2006-01-02T15:04:05+0000", e.StartTime); err == nil {
				prog.Start = t.UTC()
			} else if t, err := time.Parse("2006-01-02T15:04:05-0700", e.StartTime); err == nil {
				prog.Start = t.UTC()
			}
		}

		duration := parseDuration(e.Duration)
		if !prog.Start.IsZero() && duration > 0 {
			prog.Stop = prog.Start.Add(duration)
		}

		programs = append(programs, prog)
	}

	return programs, nil
}

var durationRegex = regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?`)

func parseDuration(s string) time.Duration {
	matches := durationRegex.FindStringSubmatch(s)
	if matches == nil {
		return 0
	}
	var d time.Duration
	if matches[1] != "" {
		h, _ := strconv.Atoi(matches[1])
		d += time.Duration(h) * time.Hour
	}
	if matches[2] != "" {
		m, _ := strconv.Atoi(matches[2])
		d += time.Duration(m) * time.Minute
	}
	if matches[3] != "" {
		sec, _ := strconv.Atoi(matches[3])
		d += time.Duration(sec) * time.Second
	}
	return d
}

type apiResponse struct {
	Data responseData `json:"data"`
}

type responseData struct {
	Programs []channelData `json:"programs"`
}

type channelData struct {
	ServiceID string  `json:"service_id"`
	Title     string  `json:"title"`
	Events    []event `json:"events"`
}

type event struct {
	ProgramID      string `json:"program_id"`
	MainTitle      string `json:"main_title"`
	SecondaryTitle string `json:"secondary_title"`
	ImageURL       string `json:"image_url"`
	StartTime      string `json:"start_time"`
	Duration       string `json:"duration"`
}
