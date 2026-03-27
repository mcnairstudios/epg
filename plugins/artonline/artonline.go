package artonline

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

var riyadh *time.Location

func init() {
	riyadh = time.FixedZone("Asia/Riyadh", 3*60*60)
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

func (p *provider) Name() string        { return "artonline.tv" }
func (p *provider) Description() string { return "ART Online TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "artonline.tv")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	parts := strings.SplitN(channel.SiteID, "#", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid site_id: %s", channel.SiteID)
	}
	siteID := parts[1]

	url := fmt.Sprintf("https://www.artonline.tv/Home/Tvlist%s", siteID)

	now := time.Now().UTC().Truncate(24 * time.Hour)
	diff := int(date.Truncate(24*time.Hour).Sub(now).Hours() / 24)

	body := fmt.Sprintf("objId=%d", diff)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel)
}

var dateRe = regexp.MustCompile(`(\d+)/(\d+)/(\d+)\s`)
var durationRe = regexp.MustCompile(`(\d+):(\d+)(?::(\d+))?`)

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	var items []apiItem
	if err := json.Unmarshal(data, &items); err != nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		start := parseStart(item)
		if start.IsZero() {
			continue
		}
		dur := parseDuration(item.Duration)
		stop := start.Add(time.Duration(dur) * time.Second)

		image := ""
		if item.Thumbnail != "" {
			image = "https://www.artonline.tv" + item.Thumbnail
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "artonline.tv",
			Title:       item.Title,
			Description: item.Description,
			Image:       image,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

func parseStart(item apiItem) time.Time {
	m := dateRe.FindStringSubmatch(item.AddDate)
	if m == nil {
		return time.Time{}
	}
	month, _ := strconv.Atoi(m[1])
	day, _ := strconv.Atoi(m[2])
	year, _ := strconv.Atoi(m[3])

	parts := strings.Split(item.StartTime, ":")
	if len(parts) < 2 {
		return time.Time{}
	}
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])

	t := time.Date(year, time.Month(month), day, hour, minute, 0, 0, riyadh)
	return t.UTC()
}

func parseDuration(dur string) int {
	m := durationRe.FindStringSubmatch(dur)
	if m == nil {
		return 0
	}
	h, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	sec := 0
	if m[3] != "" {
		sec, _ = strconv.Atoi(m[3])
	}
	return h*3600 + min*60 + sec
}

type apiItem struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Thumbnail   string `json:"thumbnail"`
	StartTime   string `json:"start_Time"`
	AddDate     string `json:"adddate"`
	Duration    string `json:"duration"`
}
