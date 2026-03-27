package ruvis

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

var reykjavikTZ = time.FixedZone("Atlantic/Reykjavik", 0)

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

func (p *provider) Name() string        { return "ruv.is" }
func (p *provider) Description() string { return "RUV Icelandic national broadcaster" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "ruv.is")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	params := url.Values{}
	params.Set("operationName", "getSchedule")
	params.Set("variables", fmt.Sprintf(`{"channel":"%s","date":"%s"}`, channel.SiteID, date.Format("2006-01-02")))
	params.Set("extensions", `{"persistedQuery":{"version":1,"sha256Hash":"7d133b9bd9e50127e90f2b3af1b41eb5e89cd386ed9b100b55169f395af350e6"}}`)

	reqURL := fmt.Sprintf("https://www.ruv.is/gql/?%s", params.Encode())

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
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
		return nil, err
	}

	if response.Data.Schedule.Events == nil {
		return []domain.Program{}, nil
	}

	dateStr := date.Format("2006-01-02")
	programs := make([]domain.Program, 0, len(response.Data.Schedule.Events))
	for _, item := range response.Data.Schedule.Events {
		start, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+item.StartTimeFriendly, reykjavikTZ)
		if err != nil {
			continue
		}
		stop, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+item.EndTimeFriendly, reykjavikTZ)
		if err != nil {
			continue
		}
		if stop.Before(start) {
			stop = stop.AddDate(0, 0, 1)
		}

		image := strings.ReplaceAll(item.Image, "$$IMAGESIZE$$", "480")

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "ruv.is",
			Title:       item.Title,
			Description: item.Description,
			Image:       image,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Data struct {
		Schedule struct {
			Events []scheduleEvent `json:"events"`
		} `json:"Schedule"`
	} `json:"data"`
}

type scheduleEvent struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	Image              string `json:"image"`
	StartTimeFriendly  string `json:"start_time_friendly"`
	EndTimeFriendly    string `json:"end_time_friendly"`
}
