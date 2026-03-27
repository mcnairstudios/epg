package magentaat

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

const apiEndpoint = "https://tv-at-prod.yo-digital.com/at-bifrost"

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

func (p *provider) Name() string        { return "tv.magenta.at" }
func (p *provider) Description() string { return "Austria Magenta TV" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tv.magenta.at")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	day := date.Truncate(24 * time.Hour).UTC()
	dateStr := day.Format("2006-01-02")

	var allPrograms []domain.Program
	var programIDs []string
	seen := make(map[string]bool)

	for _, offset := range []int{0, 3, 6, 9, 12, 15, 18, 21} {
		url := fmt.Sprintf(
			"%s/epg/channel/schedules/v2?station_ids=%s&date=%s&hour_offset=%d&hour_range=3&natco_code=at",
			apiEndpoint, channel.SiteID, dateStr, offset,
		)

		body, err := p.doGet(url)
		if err != nil {
			continue
		}

		programs, ids, err := parseScheduleWithIDs(body, channel)
		if err != nil {
			continue
		}

		for i := range programs {
			key := programs[i].Start.String() + programs[i].Title
			if seen[key] {
				continue
			}
			seen[key] = true
			allPrograms = append(allPrograms, programs[i])
			programIDs = append(programIDs, ids[i])
		}
	}

	for i, pid := range programIDs {
		detail, err := p.fetchDetail(pid)
		if err != nil || detail == nil {
			continue
		}
		if detail.Details != nil {
			allPrograms[i].Description = detail.Details.Description
		}
		if detail.PosterImageURL != "" {
			allPrograms[i].Image = detail.PosterImageURL
		}
	}

	return allPrograms, nil
}

func (p *provider) fetchDetail(programID string) (*DetailResponse, error) {
	if programID == "" {
		return nil, nil
	}
	url := fmt.Sprintf("%s/details/series/%s?natco_code=at", apiEndpoint, programID)
	body, err := p.doGet(url)
	if err != nil {
		return nil, err
	}
	return ParseDetail(body)
}

func (p *provider) doGet(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Device-Id", randomUUID())
	req.Header.Set("app_key", "CTnKA63ruKM0JM1doxAXwwyQLLmQiEiy")
	req.Header.Set("app_version", "02.0.1260")
	req.Header.Set("X-User-Agent", "web|web|Firefox-120|02.0.1260|1")
	req.Header.Set("x-request-tracking-id", randomUUID())

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	programs, _, err := parseScheduleWithIDs(data, channel)
	return programs, err
}

func parseScheduleWithIDs(data []byte, channel domain.Channel) ([]domain.Program, []string, error) {
	var response scheduleResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil, err
	}

	items, ok := response.Channels[channel.SiteID]
	if !ok {
		return nil, nil, nil
	}

	programs := make([]domain.Program, 0, len(items))
	ids := make([]string, 0, len(items))
	for _, item := range items {
		p := domain.Program{
			Channel:  channel.XmltvID,
			Source:   "tv.magenta.at",
			Title:    item.Description,
			Lang:     channel.Lang,
			Category: parseCategory(item.Genres),
			Season:   parseSeason(item),
			Episode:  parseEpisode(item),
		}

		if item.StartTime != "" {
			if t, err := time.Parse("2006-01-02T15:04:05.00Z", item.StartTime); err == nil {
				p.Start = t
			}
		}
		if item.EndTime != "" {
			if t, err := time.Parse("2006-01-02T15:04:05.00Z", item.EndTime); err == nil {
				p.Stop = t
			}
		}

		programs = append(programs, p)
		ids = append(ids, item.ProgramID)
	}

	return programs, ids, nil
}

func ParseDetail(data []byte) (*DetailResponse, error) {
	var detail DetailResponse
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

func parseCategory(genres []genre) string {
	if len(genres) == 0 {
		return ""
	}
	ids := make([]string, 0, len(genres))
	for _, g := range genres {
		ids = append(ids, g.ID)
	}
	return strings.Join(ids, ",")
}

func parseSeason(item scheduleItem) int {
	if item.SeasonDisplayNumber == "Folgen" {
		return 0
	}
	return item.SeasonNumber
}

func parseEpisode(item scheduleItem) int {
	if item.EpisodeNumber != "" {
		n, _ := strconv.Atoi(item.EpisodeNumber)
		return n
	}
	if item.SeasonDisplayNumber == "Folgen" {
		return item.SeasonNumber
	}
	return 0
}

type scheduleResponse struct {
	Channels map[string][]scheduleItem `json:"channels"`
}

type scheduleItem struct {
	ProgramID           string  `json:"program_id"`
	Description         string  `json:"description"`
	StartTime           string  `json:"start_time"`
	EndTime             string  `json:"end_time"`
	Genres              []genre `json:"genres"`
	ReleaseYear         int     `json:"release_year"`
	SeasonNumber        int     `json:"season_number"`
	SeasonDisplayNumber string  `json:"season_display_number"`
	EpisodeNumber       string  `json:"episode_number"`
}

type genre struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DetailResponse struct {
	PosterImageURL string      `json:"poster_image_url"`
	Details        *detailInfo `json:"details"`
	Roles          []Role      `json:"roles"`
}

type detailInfo struct {
	Description string `json:"description"`
}

type Role struct {
	RoleName   string `json:"role_name"`
	PersonName string `json:"person_name"`
}

func randomUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
