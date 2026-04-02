package pbsguam

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
	tlsClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	plugin.Register(&provider{
		client: tlsClient,
	})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "pbsguam.org" }
func (p *provider) Description() string { return "PBS Guam" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "pbsguam.org")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	req, err := http.NewRequest(http.MethodGet, "https://pbsguam.org/calendar/", nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from pbsguam", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

var scheduleRe = regexp.MustCompile(`(?i)EventsSchedule_1 = (.*);`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	matches := scheduleRe.FindSubmatch(data)
	if matches == nil || len(matches) < 2 {
		return []domain.Program{}, nil
	}

	var schedule scheduleData
	if err := json.Unmarshal(matches[1], &schedule); err != nil {
		return []domain.Program{}, nil
	}

	if schedule.Feed == nil {
		return []domain.Program{}, nil
	}

	d := date.Truncate(24 * time.Hour).UTC()
	nextDay := d.Add(24 * time.Hour)

	programs := make([]domain.Program, 0)
	for _, item := range schedule.Feed {
		startT, err := time.Parse(time.RFC3339, item.Start)
		if err != nil {
			continue
		}
		startT = startT.UTC()
		endT, err := time.Parse(time.RFC3339, item.End)
		if err != nil {
			continue
		}
		endT = endT.UTC()

		startBetween := (startT.Equal(d) || startT.After(d)) && startT.Before(nextDay)
		endBetween := (endT.Equal(d) || endT.After(d)) && endT.Before(nextDay)

		if !startBetween && !endBetween {
			continue
		}

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "pbsguam.org",
			Title:   item.Title,
			Lang:    channel.Lang,
			Start:   startT,
			Stop:    endT,
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type scheduleData struct {
	Feed []feedItem `json:"feed"`
}

type feedItem struct {
	Title string `json:"title"`
	Start string `json:"start"`
	End   string `json:"end"`
}

func (p *provider) Timezone() string { return "Pacific/Guam" }
