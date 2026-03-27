package astro

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

func (p *provider) Name() string        { return "content.astro.com.my" }
func (p *provider) Description() string { return "Astro Malaysia TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "content.astro.com.my")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://contenthub-api.eco.astro.com.my/channel/%s.json", channel.SiteID)

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
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil
	}

	dateKey := date.Format("2006-01-02")
	items, ok := response.Response.Schedule[dateKey]
	if !ok {
		return nil, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		start, _ := time.Parse("2006-01-02 15:04:05.0", item.DatetimeInUTC)
		start = start.UTC()
		dur := parseDurationHMS(item.Duration)
		stop := start.Add(dur)

		prog := domain.Program{
			Channel: channel.XmltvID,
			Source:  "content.astro.com.my",
			Title:   item.Title,
			Lang:    channel.Lang,
			Start:   start,
			Stop:    stop,
		}

		ep := parseEpisodeFromTitle(item.Title)
		if ep > 0 {
			prog.Episode = ep
		}

		programs = append(programs, prog)
	}

	return programs, nil
}

var epRegex = regexp.MustCompile(`Ep(\d+)$`)

func parseEpisodeFromTitle(title string) int {
	matches := epRegex.FindStringSubmatch(title)
	if matches == nil {
		return 0
	}
	n, _ := strconv.Atoi(matches[1])
	return n
}

func parseDurationHMS(duration string) time.Duration {
	parts := strings.Split(duration, ":")
	if len(parts) != 3 {
		return 0
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	s, _ := strconv.Atoi(parts[2])
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s)*time.Second
}

type apiResponse struct {
	Response responseBody `json:"response"`
}

type responseBody struct {
	Schedule map[string][]scheduleItem `json:"schedule"`
}

type scheduleItem struct {
	EventID       string `json:"eventId"`
	Title         string `json:"title"`
	DatetimeInUTC string `json:"datetimeInUtc"`
	Duration      string `json:"duration"`
	SiTrafficKey  string `json:"siTrafficKey"`
}
