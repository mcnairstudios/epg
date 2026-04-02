package rtppt

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
	plugin.Register(&provider{client: httpclient.Browser()})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "rtp.pt" }
func (p *provider) Description() string { return "RTP Portugal EPG" }

var timezones = map[string]string{
	"lis": "Europe/Lisbon",
	"per": "Asia/Macau",
	"rja": "America/Sao_Paulo",
}

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "rtp.pt")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	parts := strings.SplitN(channel.SiteID, "#", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid site_id format: %s", channel.SiteID)
	}
	region := parts[0]
	channelCode := parts[1]

	url := fmt.Sprintf(
		"https://www.rtp.pt/EPG/json/rtp-channels-page/list-grid/tv/%s/%s/%s",
		channelCode,
		fmt.Sprintf("%d-%d-%d", date.Day(), int(date.Month()), date.Year()),
		region,
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
		return nil, fmt.Errorf("HTTP %d from rtppt", resp.StatusCode)
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

	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return []domain.Program{}, nil
	}

	parts := strings.SplitN(channel.SiteID, "#", 2)
	region := "lis"
	if len(parts) >= 1 {
		region = parts[0]
	}

	tzName, ok := timezones[region]
	if !ok {
		tzName = "Europe/Lisbon"
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		loc = time.UTC
	}

	var allItems []epgItem
	for _, items := range resp.Result {
		allItems = append(allItems, items...)
	}

	programs := make([]domain.Program, 0, len(allItems))
	for i, item := range allItems {
		start, err := time.ParseInLocation("2006-01-02 15:04:05", item.Date, loc)
		if err != nil {
			continue
		}

		stop := start.Add(30 * time.Minute)
		if i+1 < len(allItems) {
			nextStart, err := time.ParseInLocation("2006-01-02 15:04:05", allItems[i+1].Date, loc)
			if err == nil {
				stop = nextStart
			}
		}

		var image string
		if len(item.Image) > 0 {
			image = item.Image[len(item.Image)-1].Src
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "rtp.pt",
			Title:       item.Name,
			Description: item.Description,
			Image:       image,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		})
	}

	return programs, nil
}

type apiResponse struct {
	Result map[string][]epgItem `json:"result"`
}

type epgItem struct {
	Date        string      `json:"date"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Image       []imageItem `json:"image"`
}

type imageItem struct {
	Src string `json:"src"`
}

func (p *provider) Timezone() string { return "Europe/Lisbon" }
