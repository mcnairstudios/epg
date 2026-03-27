package derana2

import (
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

var colomboTZ = time.FixedZone("Asia/Colombo", 5*60*60+30*60)

//go:embed channels.json
var channelsData []byte

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

func (p *provider) Name() string        { return "derana.lk.v2" }
func (p *provider) Description() string { return "TV Derana Sri Lanka (v2)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "derana.lk.v2")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://derana.lk/api/schedules/%s", date.Format("02-01-2006"))

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
		return nil, err
	}

	programs := make([]domain.Program, 0, len(response.AllSchedules))
	for _, item := range response.AllSchedules {
		start := parseStart(item.Date, item.Time)
		if start.IsZero() {
			continue
		}

		duration := parseDuration(item.Duration)
		stop := start.Add(duration)

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "derana.lk.v2",
			Title:   item.DramaName,
			Image:   item.ImageURL,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func parseStart(dateStr, timeStr string) time.Time {
	t, err := time.ParseInLocation("02-01-2006 3:04 PM", dateStr+" "+timeStr, colomboTZ)
	if err != nil {
		return time.Time{}
	}
	return t
}

func parseDuration(s string) time.Duration {
	s = strings.TrimSpace(s)
	parts := strings.Fields(s)
	var total time.Duration
	for i := 0; i < len(parts)-1; i += 2 {
		val, err := strconv.Atoi(parts[i])
		if err != nil {
			continue
		}
		unit := strings.ToLower(parts[i+1])
		switch {
		case strings.HasPrefix(unit, "hour"):
			total += time.Duration(val) * time.Hour
		case strings.HasPrefix(unit, "min"):
			total += time.Duration(val) * time.Minute
		}
	}
	return total
}

type apiResponse struct {
	AllSchedules []scheduleItem `json:"all_schedules"`
}

type scheduleItem struct {
	DramaName string `json:"dramaName"`
	ImageURL  string `json:"imageUrl"`
	Time      string `json:"time"`
	Duration  string `json:"duration"`
	Date      string `json:"date"`
}
