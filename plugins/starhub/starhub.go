package starhub

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
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{client: httpclient.Browser()})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "starhubtvplus.com" }
func (p *provider) Description() string { return "StarHub TV+ EPG service" }

var languages = map[string]string{
	"en": "en_US",
	"zh": "zh",
}

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "starhubtvplus.com")
}

func buildURL(channel domain.Channel, date time.Time, page int) string {
	locale := languages[channel.Lang]
	if locale == "" {
		locale = "en_US"
	}
	start := date.Truncate(24 * time.Hour).UTC()
	end := start.Add(24 * time.Hour)
	return fmt.Sprintf(
		"https://waf-starhub-metadata-api-p001.ifs.vubiquity.com/v3.1/epg/schedules?locale=%s&locale_default=%s&device=1&in_channel_id=%s&gt_end=%d&lt_start=%d&limit=100&page=%d",
		locale, locale, channel.SiteID, start.Unix(), end.Unix(), page,
	)
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	var allResources []resource
	page := 1
	for {
		url := buildURL(channel, date, page)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}

		resp, err := p.client.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusNoContent || len(body) == 0 {
			return []domain.Program{}, nil
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
		}

		var ar apiResponse
		if err := json.Unmarshal(body, &ar); err != nil {
			return nil, err
		}

		allResources = append(allResources, ar.Resources...)

		if ar.Page.Current >= ar.Page.Total {
			break
		}
		page = ar.Page.Current + 1
	}

	return convert(allResources, channel), nil
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var ar apiResponse
	if err := json.Unmarshal(data, &ar); err != nil {
		return nil, err
	}
	return convert(ar.Resources, channel), nil
}

var seasonRegex = regexp.MustCompile(`(?:S|Season )(\d+)`)

func parseSeason(serieTitle string) int {
	if serieTitle == "" {
		return 0
	}
	m := seasonRegex.FindStringSubmatch(serieTitle)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

func convert(resources []resource, channel domain.Channel) []domain.Program {
	programs := make([]domain.Program, 0, len(resources))
	for _, r := range resources {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "starhubtvplus.com",
			Title:       r.Title,
			SubTitle:    r.SerieTitle,
			Description: r.Description,
			Season:      parseSeason(r.SerieTitle),
			Episode:     r.EpisodeNumber,
			Lang:        channel.Lang,
			Start:       time.Unix(int64(r.Start), 0).UTC(),
			Stop:        time.Unix(int64(r.End), 0).UTC(),
		}

		if len(r.Genres) > 0 {
			p.Category = strings.Join(r.Genres, ",")
		}

		if r.Rating != "" {
			p.Description = r.Description
		}

		if len(r.Pictures) > 0 {
			p.Image = r.Pictures[0].URL
		}

		programs = append(programs, p)
	}
	return programs
}

type apiResponse struct {
	Resources []resource   `json:"resources"`
	Page      pageInfo     `json:"page"`
}

type pageInfo struct {
	Current int `json:"current"`
	Total   int `json:"total"`
}

type resource struct {
	Title         string    `json:"title"`
	SerieTitle    string    `json:"serie_title"`
	Description   string    `json:"description"`
	Rating        string    `json:"rating"`
	EpisodeNumber int       `json:"episode_number"`
	Start         int64     `json:"start"`
	End           int64     `json:"end"`
	Genres        []string  `json:"genres"`
	Pictures      []picture `json:"pictures"`
}

type picture struct {
	URL  string `json:"url"`
	Type int    `json:"type"`
}
