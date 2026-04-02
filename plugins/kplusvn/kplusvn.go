package kplusvn

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

func (p *provider) Name() string        { return "xem.kplus.vn" }
func (p *provider) Description() string { return "K+ Vietnam TV listings" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "xem.kplus.vn")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour).UTC()

	url := fmt.Sprintf(
		"https://tvapi-sgn.solocoo.tv/v1/assets?query=schedule,forrelated,%s&from=%s&limit=1000",
		channel.SiteID,
		start.Format("2006-01-02T15:04:05Z"),
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
		return nil, fmt.Errorf("HTTP %d from kplusvn", resp.StatusCode)
	}

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

	if response.Assets == nil {
		return nil, nil
	}

	programs := make([]domain.Program, 0, len(response.Assets))
	for _, item := range response.Assets {
		if item.Params == nil || item.Params.Start == "" {
			continue
		}

		startTime, err := time.Parse("2006-01-02T15:04:05Z", item.Params.Start)
		if err != nil {
			continue
		}

		if startTime.Year() != date.Year() || startTime.YearDay() != date.YearDay() {
			continue
		}

		var stopTime time.Time
		if item.Params.End != "" {
			stopTime, _ = time.Parse("2006-01-02T15:04:05Z", item.Params.End)
		}

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "xem.kplus.vn",
			Title:   item.Title,
			Lang:    channel.Lang,
			Start:   startTime,
			Stop:    stopTime,
		}

		if item.Params.Genres != nil {
			cats := make([]string, 0, len(item.Params.Genres))
			for _, g := range item.Params.Genres {
				cats = append(cats, g.Title)
			}
			if len(cats) > 0 {
				p.Category = strings.Join(cats, ", ")
			}
		}

		if item.Images != nil {
			for _, img := range item.Images {
				if strings.Contains(img.URL, "orientation=landscape") {
					p.Image = img.URL + "&w=460&h=260"
					break
				}
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Assets []asset `json:"assets"`
}

type asset struct {
	Title  string      `json:"title"`
	Params *assetParam `json:"params"`
	Images []image     `json:"images"`
}

type assetParam struct {
	Start  string  `json:"start"`
	End    string  `json:"end"`
	Genres []genre `json:"genres"`
}

type genre struct {
	Title string `json:"title"`
}

type image struct {
	URL string `json:"url"`
}

func (p *provider) Timezone() string { return "Asia/Ho_Chi_Minh" }
