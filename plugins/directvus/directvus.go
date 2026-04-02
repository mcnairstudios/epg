package directvus

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

func (p *provider) Name() string        { return "directv.com" }
func (p *provider) Description() string { return "DirecTV US satellite television" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "directv.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	parts := strings.SplitN(channel.SiteID, "#", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid site_id %q: expected format channelNum#childId", channel.SiteID)
	}
	channelNum := parts[0]
	childID := parts[1]

	start := date.Truncate(24 * time.Hour).UTC()
	url := fmt.Sprintf(
		"https://www.directv.com/json/channelschedule?channels=%s&startTime=%s&hours=24&chId=%s",
		channelNum,
		start.Format(time.RFC3339),
		childID,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	items, err := extractSchedules(body, childID)
	if err != nil {
		return nil, err
	}

	details := make(map[string]*ProgramDetail)
	for _, item := range items {
		if item.ProgramID == "-1" || item.ProgramID == "" {
			continue
		}
		detail, _ := p.fetchDetail(item.ProgramID)
		if detail != nil {
			details[item.ProgramID] = detail
		}
	}

	return Parse(body, channel, details)
}

func (p *provider) fetchDetail(programID string) (*ProgramDetail, error) {
	url := fmt.Sprintf("https://www.directv.com/json/program/flip/%s", programID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d fetching program detail %s", resp.StatusCode, programID)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var flipResp flipResponse
	if err := json.Unmarshal(body, &flipResp); err != nil {
		return nil, err
	}

	return flipResp.ProgramDetail, nil
}

func Parse(data []byte, channel domain.Channel, details map[string]*ProgramDetail) ([]domain.Program, error) {
	parts := strings.SplitN(channel.SiteID, "#", 2)
	childID := ""
	if len(parts) == 2 {
		childID = parts[1]
	}

	items, err := extractSchedules(data, childID)
	if err != nil {
		return nil, err
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		if item.ProgramID == "-1" {
			continue
		}

		start, err := parseAirTime(item.AirTime)
		if err != nil {
			continue
		}
		stop := start.Add(time.Duration(item.Duration) * time.Minute)

		p := domain.Program{
			Channel:  channel.XmltvID,
			Source:   "directv.com",
			Title:    item.Title,
			SubTitle: item.EpisodeTitle,
			Lang:     channel.Lang,
			Start:    start,
			Stop:     stop,
		}

		if item.SeasonNumber > 0 {
			p.Season = item.SeasonNumber
		}
		if item.EpisodeNumber > 0 {
			p.Episode = item.EpisodeNumber
		}

		if item.PrimaryImageURL != "" {
			p.Image = "https://www.directv.com" + item.PrimaryImageURL
		}

		if len(item.SubcategoryList) > 0 {
			p.Category = strings.Join(item.SubcategoryList, ", ")
		}

		if detail, ok := details[item.ProgramID]; ok && detail != nil {
			if detail.Description != "" {
				p.Description = detail.Description
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func extractSchedules(data []byte, childID string) ([]scheduleItem, error) {
	var response scheduleResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	if len(response.Schedule) == 0 {
		return nil, nil
	}

	for _, ch := range response.Schedule {
		if fmt.Sprintf("%v", ch.ChID) == childID {
			return ch.Schedules, nil
		}
	}

	return nil, nil
}

func parseAirTime(airTime string) (time.Time, error) {
	layouts := []string{
		"2006-01-02T15:04:05.000+0000",
		"2006-01-02T15:04:05.000-0700",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, airTime); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse air time %q", airTime)
}

type scheduleResponse struct {
	Schedule []channelSchedule `json:"schedule"`
}

type channelSchedule struct {
	ChID      json.Number    `json:"chId"`
	ChNum     int            `json:"chNum"`
	Schedules []scheduleItem `json:"schedules"`
}

type scheduleItem struct {
	Title           string   `json:"title"`
	EpisodeTitle    string   `json:"episodeTitle"`
	ProgramID       string   `json:"programID"`
	AirTime         string   `json:"airTime"`
	Duration        int      `json:"duration"`
	Rating          string   `json:"rating"`
	SeasonNumber    int      `json:"seasonNumber"`
	EpisodeNumber   int      `json:"episodeNumber"`
	PrimaryImageURL string   `json:"primaryImageUrl"`
	SubcategoryList []string `json:"subcategoryList"`
}

type flipResponse struct {
	ProgramDetail *ProgramDetail `json:"programDetail"`
}

type ProgramDetail struct {
	Description string `json:"description"`
	ReleaseYear string `json:"releaseYear"`
}

func (p *provider) Timezone() string { return "America/New_York" }
