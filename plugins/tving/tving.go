package tving

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

var seoulTZ *time.Location

//go:embed channels.json
var channelsData []byte

var jsonpRe = regexp.MustCompile(`^cb\((.*)\);?$`)

func init() {
	seoulTZ, _ = time.LoadLocation("Asia/Seoul")
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

func (p *provider) Name() string        { return "m.tving.com" }
func (p *provider) Description() string { return "Tving Korean streaming service" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "m.tving.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://api.tving.com/v2/media/schedules/%s/%s?callback=cb&pageNo=1&pageSize=500&screenCode=CSSD0200&networkCode=CSND0900&osCode=CSOD0900&teleCode=CSCD0900&apiKey=4263d7d76161f4a19a9efe9ca7903ec4",
		channel.SiteID,
		date.Format("20060102"),
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from tving", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	jsonData := extractJSON(string(data))
	if jsonData == "" {
		return []domain.Program{}, nil
	}

	var response apiResponse
	if err := json.Unmarshal([]byte(jsonData), &response); err != nil {
		return []domain.Program{}, nil
	}

	if response.Body == nil || len(response.Body.Result) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(response.Body.Result))
	for _, item := range response.Body.Result {
		start := parseTime(item.BroadcastStartTime)
		stop := parseTime(item.BroadcastEndTime)
		if start.IsZero() || stop.IsZero() {
			continue
		}

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "m.tving.com",
			Title:   item.Program.Name.Ko,
			Lang:    channel.Lang,
			Start:   start,
			Stop:    stop,
		}

		if item.Program.Synopsis.Ko != "" {
			p.Description = item.Program.Synopsis.Ko
		}

		if item.Program.ProductYear > 0 {
			p.SubTitle = fmt.Sprintf("%d", item.Program.ProductYear)
		}

		if len(item.Program.Image) > 0 && item.Program.Image[0].URL != "" {
			p.Image = "https://image.tving.com" + item.Program.Image[0].URL
		}

		categories := buildCategories(item)
		if len(categories) > 0 {
			p.Category = categories[0]
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func extractJSON(content string) string {
	matches := jsonpRe.FindStringSubmatch(content)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func parseTime(ts int64) time.Time {
	if ts == 0 {
		return time.Time{}
	}
	s := fmt.Sprintf("%d", ts)
	if len(s) != 14 {
		return time.Time{}
	}
	t, err := time.ParseInLocation("20060102150405", s, seoulTZ)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func buildCategories(item scheduleItem) []string {
	var cats []string
	if item.Category1Name != nil && item.Category1Name.Ko != "" {
		cats = append(cats, item.Category1Name.Ko)
	}
	if item.Category2Name != nil && item.Category2Name.Ko != "" {
		cats = append(cats, item.Category2Name.Ko)
	}
	return cats
}

type apiResponse struct {
	Body *apiBody `json:"body"`
}

type apiBody struct {
	Result []scheduleItem `json:"result"`
}

type scheduleItem struct {
	BroadcastStartTime int64       `json:"broadcast_start_time"`
	BroadcastEndTime   int64       `json:"broadcast_end_time"`
	Category1Name      *localeName `json:"category1_name"`
	Category2Name      *localeName `json:"category2_name"`
	Program            programInfo `json:"program"`
}

type programInfo struct {
	Name        localeName   `json:"name"`
	Synopsis    localeName   `json:"synopsis"`
	ProductYear int          `json:"product_year"`
	Director    interface{}  `json:"director"`
	Actor       interface{}  `json:"actor"`
	Image       []imageEntry `json:"image"`
}

type localeName struct {
	Ko string `json:"ko"`
}

type imageEntry struct {
	URL string `json:"url"`
}

func (p *provider) Timezone() string { return "Asia/Seoul" }
