package playeree

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

var seasonEpisodeRe = regexp.MustCompile(`\(?S(\d+)[/\s]Ep(\d+)\)?`)

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

func (p *provider) Name() string        { return "player.ee.co.uk" }
func (p *provider) Description() string { return "UK EE TV" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "player.ee.co.uk")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	day := date.Truncate(24 * time.Hour).UTC()

	first, err := p.fetchWindow(channel.SiteID, day, 0)
	if err != nil {
		return nil, err
	}

	second, err := p.fetchWindow(channel.SiteID, day, 12)
	if err != nil {
		return nil, err
	}

	items := append(first, second...)
	return Parse(items, channel)
}

func (p *provider) fetchWindow(siteID string, day time.Time, hour int) ([]ScheduleItem, error) {
	u := buildURL(siteID, day, hour)

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", "https://player.ee.co.uk/")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var sr scheduleResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return nil, err
	}

	return sr.Items, nil
}

func buildURL(siteID string, day time.Time, hour int) string {
	return fmt.Sprintf(
		"https://api.youview.tv/metadata/linear/v2/schedule/by-servicelocator?serviceLocator=%s&interval=%sT%02dZ/PT12H",
		url.QueryEscape(siteID),
		day.Format("2006-01-02"),
		hour,
	)
}

func ParseResponse(data []byte) ([]ScheduleItem, error) {
	var sr scheduleResponse
	if err := json.Unmarshal(data, &sr); err != nil {
		return nil, err
	}
	return sr.Items, nil
}

func Parse(items []ScheduleItem, channel domain.Channel) ([]domain.Program, error) {
	programs := make([]domain.Program, 0, len(items))

	for _, item := range items {
		start, err := time.Parse(time.RFC3339, item.PublishedStartTime)
		if err != nil {
			continue
		}
		stop := start.Add(time.Duration(item.PublishedDuration) * time.Second)

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "player.ee.co.uk",
			Title:       item.Title,
			Description: item.Synopsis,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		}

		if item.Synopsis != "" {
			if matches := seasonEpisodeRe.FindStringSubmatch(item.Synopsis); matches != nil {
				if s, err := strconv.Atoi(matches[1]); err == nil {
					p.Season = s
				}
				if e, err := strconv.Atoi(matches[2]); err == nil {
					p.Episode = e
				}
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type scheduleResponse struct {
	Items []ScheduleItem `json:"items"`
}

type ScheduleItem struct {
	Title              string `json:"title"`
	Synopsis           string `json:"synopsis"`
	PublishedStartTime string `json:"publishedStartTime"`
	PublishedDuration  int    `json:"publishedDuration"`
}
