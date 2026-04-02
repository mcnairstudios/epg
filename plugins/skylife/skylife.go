package skylife

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

var seoulTZ = time.FixedZone("Asia/Seoul", 9*60*60)

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

func (p *provider) Name() string        { return "skylife.co.kr" }
func (p *provider) Description() string { return "SkyLife Korean satellite TV" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "skylife.co.kr")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://www.skylife.co.kr/api/api/public/tv/schedule/%s", date.Format("20060102"))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from skylife", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var categories []category
	if err := json.Unmarshal(data, &categories); err != nil {
		return nil, err
	}

	parts := strings.SplitN(channel.SiteID, "#", 2)
	if len(parts) != 2 {
		return []domain.Program{}, nil
	}
	categoryCode := parts[0]
	channelID := parts[1]

	var items []programItem
	for _, cat := range categories {
		if cat.Code != categoryCode {
			continue
		}
		for _, ch := range cat.Channels {
			if ch.ID != channelID {
				continue
			}
			items = ch.Programs
			break
		}
		break
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		start := parseTime(item.StartTime)
		stop := parseTime(item.EndTime)
		if start.IsZero() {
			continue
		}

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "skylife.co.kr",
			Title:       item.Name,
			Description: item.Summary,
			Category:    item.MainCategory,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func parseTime(s string) time.Time {
	t, err := time.ParseInLocation("20060102150405", s, seoulTZ)
	if err != nil {
		return time.Time{}
	}
	return t
}

type category struct {
	Code     string        `json:"code"`
	Channels []channelItem `json:"channels"`
}

type channelItem struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Programs []programItem `json:"programs"`
}

type programItem struct {
	Name         string `json:"name"`
	MainCategory string `json:"mainCategory"`
	Summary      string `json:"summary"`
	StartTime    string `json:"startTime"`
	EndTime      string `json:"endTime"`
}

func (p *provider) Timezone() string { return "Asia/Seoul" }
