package firstmedia

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
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

func (p *provider) Name() string        { return "firstmedia.com" }
func (p *provider) Description() string { return "First Media Indonesia TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "firstmedia-243", Name: "Al Jazeera International", Source: "firstmedia.com", SiteID: "243", Lang: "en", XmltvID: "AlJazeeraEnglish.qa"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Format("02/01/2006")
	url := fmt.Sprintf(
		"https://api.firstmedia.com/api/content/tv-guide/list?date=%s&channel=%s&startTime=1&endTime=24",
		d, channel.SiteID,
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

	return Parse(body, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return []domain.Program{}, nil
	}

	entries, ok := response.Data.Entries[channel.SiteID]
	if !ok || len(entries) == 0 {
		return []domain.Program{}, nil
	}

	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, jakarta)

	type parsed struct {
		entry    entry
		startDt  time.Duration
		stopDt   time.Duration
	}

	var items []parsed
	for _, e := range entries {
		baseTime, err := time.ParseInLocation("2006-01-02 15:04:05", e.Date, jakarta)
		if err != nil {
			continue
		}
		startTime, err := time.ParseInLocation("2006-01-02 15:04:05", e.StartTime, jakarta)
		if err != nil {
			continue
		}
		endTime, err := time.ParseInLocation("2006-01-02 15:04:05", e.EndTime, jakarta)
		if err != nil {
			continue
		}

		startDelta := startTime.Sub(baseTime)
		stopDelta := endTime.Sub(baseTime)

		items = append(items, parsed{
			entry:   e,
			startDt: startDelta,
			stopDt:  stopDelta,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].startDt < items[j].startDt
	})

	programs := make([]domain.Program, 0, len(items))
	var lastStop time.Duration
	first := true
	for _, item := range items {
		if !first && item.startDt < lastStop {
			continue
		}
		first = false
		lastStop = item.stopDt

		start := dayStart.Add(item.startDt).UTC()
		stop := dayStart.Add(item.stopDt).UTC()

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "firstmedia.com",
			Title:       item.entry.Title,
			Description: item.entry.LongDescription,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

type apiResponse struct {
	Data responseData `json:"data"`
}

type responseData struct {
	Entries map[string][]entry `json:"entries"`
}

type entry struct {
	Title           string `json:"title"`
	LongDescription string `json:"long_description"`
	Date            string `json:"date"`
	StartTime       string `json:"startTime"`
	EndTime         string `json:"endTime"`
}
