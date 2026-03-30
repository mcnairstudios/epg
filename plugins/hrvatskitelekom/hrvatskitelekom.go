package hrvatskitelekom

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

const (
	natcoCode  = "hr"
	appKey     = "GWaBW4RTloLwpUgYVzOiW5zUxFLmoMj5"
	appVersion = "02.0.1080"
	siteURL    = "mojmaxtv.hrvatskitelekom.hr"
)

var apiEndpoint = fmt.Sprintf("https://tv-%s-prod.yo-digital.com/%s-bifrost", natcoCode, natcoCode)

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

func (p *provider) Name() string        { return siteURL }
func (p *provider) Description() string { return "HRT MaxTV Croatia (243 channels)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, siteURL)
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	dateStr := date.Format("2006-01-02")
	var allItems []scheduleItem

	for _, offset := range []int{0, 3, 6, 9, 12, 15, 18, 21} {
		url := fmt.Sprintf(
			"%s/epg/channel/schedules?date=%s&hour_offset=%d&hour_range=3&channelMap_id&filler=true&app_language=%s&natco_code=%s",
			apiEndpoint, dateStr, offset, natcoCode, natcoCode,
		)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		setHeaders(req)

		resp, err := p.client.Do(req)
		if err != nil {
			continue
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		items := parseScheduleItems(data, channel.SiteID)
		allItems = append(allItems, items...)
	}

	return buildPrograms(allItems, channel), nil
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	items := parseScheduleItems(data, channel.SiteID)
	return buildPrograms(items, channel), nil
}

func ParseMulti(datasets [][]byte, channel domain.Channel) ([]domain.Program, error) {
	var allItems []scheduleItem
	for _, data := range datasets {
		items := parseScheduleItems(data, channel.SiteID)
		allItems = append(allItems, items...)
	}
	return buildPrograms(allItems, channel), nil
}

func buildPrograms(items []scheduleItem, channel domain.Channel) []domain.Program {
	sort.Slice(items, func(i, j int) bool {
		return items[i].StartTime < items[j].StartTime
	})

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		p := domain.Program{
			Channel:  channel.XmltvID,
			Source:   siteURL,
			Title:    item.Description,
			SubTitle: item.EpisodeName,
			Lang:     channel.Lang,
		}

		if len(item.Genres) > 0 {
			names := make([]string, 0, len(item.Genres))
			for _, g := range item.Genres {
				names = append(names, g.Name)
			}
			if len(names) > 0 {
				p.Category = names[0]
			}
		}

		if item.ReleaseYear != nil {
			p.Description = ""
		}

		p.Season = parseSeason(item)
		p.Episode = parseEpisode(item)

		if t, err := time.Parse("2006-01-02T15:04:05.00Z", item.StartTime); err == nil {
			p.Start = t
		} else if t, err := time.Parse(time.RFC3339, item.StartTime); err == nil {
			p.Start = t
		}

		if t, err := time.Parse("2006-01-02T15:04:05.00Z", item.EndTime); err == nil {
			p.Stop = t
		} else if t, err := time.Parse(time.RFC3339, item.EndTime); err == nil {
			p.Stop = t
		}

		programs = append(programs, p)
	}

	return programs
}

func parseSeason(item scheduleItem) int {
	if item.SeasonDisplayNumber != nil && *item.SeasonDisplayNumber == "Epizode" {
		return 0
	}
	if item.SeasonNumber != nil {
		return *item.SeasonNumber
	}
	return 0
}

func parseEpisode(item scheduleItem) int {
	ep := item.episodeNum()
	if ep != nil {
		return *ep
	}
	if item.SeasonDisplayNumber != nil && *item.SeasonDisplayNumber == "Epizode" && item.SeasonNumber != nil {
		return *item.SeasonNumber
	}
	return 0
}

func parseScheduleItems(data []byte, stationID string) []scheduleItem {
	var response scheduleResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil
	}

	items, ok := response.Channels[stationID]
	if !ok {
		return nil
	}
	return items
}

func setHeaders(req *http.Request) {
	uuid := randomUUID()
	req.Header.Set("app_key", appKey)
	req.Header.Set("app_version", appVersion)
	req.Header.Set("device-id", randomUUID())
	req.Header.Set("tenant", "tv")
	req.Header.Set("origin", "https://"+siteURL)
	req.Header.Set("x-request-session-id", randomUUID())
	req.Header.Set("x-request-tracking-id", uuid)
	req.Header.Set("x-tv-step", "EPG_SCHEDULES")
	req.Header.Set("x-tv-flow", "EPG")
	req.Header.Set("x-call-type", "GUEST_USER")
	req.Header.Set("x-user-agent", fmt.Sprintf("web|web|Chrome-133|%s|1", appVersion))
}

func randomUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
}

type scheduleResponse struct {
	Channels map[string][]scheduleItem `json:"channels"`
}

type scheduleItem struct {
	ProgramID            string          `json:"program_id"`
	Description          string          `json:"description"`
	EpisodeName          string          `json:"episode_name"`
	StartTime            string          `json:"start_time"`
	EndTime              string          `json:"end_time"`
	Genres               []genreEntry    `json:"genres"`
	ReleaseYear          *int            `json:"release_year"`
	SeasonNumber         *int            `json:"season_number"`
	SeasonDisplayNumber  *string         `json:"season_display_number"`
	EpisodeNumber        json.RawMessage `json:"episode_number"`
	Ratings              string          `json:"ratings"`
	PosterImageURL       *string         `json:"poster_image_url"`
}

func (s scheduleItem) episodeNum() *int {
	if len(s.EpisodeNumber) == 0 || string(s.EpisodeNumber) == "null" {
		return nil
	}
	var n int
	if err := json.Unmarshal(s.EpisodeNumber, &n); err == nil {
		return &n
	}
	var str string
	if err := json.Unmarshal(s.EpisodeNumber, &str); err == nil {
		fmt.Sscanf(str, "%d", &n)
		return &n
	}
	return nil
}

type genreEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
