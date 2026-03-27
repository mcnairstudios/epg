package airtelxstream

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
	"github.com/gavinmcnair/epg/internal/plugin"
)

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

func (p *provider) Name() string        { return "airtelxstream.in" }
func (p *provider) Description() string { return "Airtel Xstream India live TV EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "airtelxstream.in")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour)
	end := start.Add(24 * time.Hour)

	url := fmt.Sprintf(
		"https://epg.airtel.tv/app/v2/content/channel/epg?channelId=%s&startTime=%d&endTime=%d",
		channel.SiteID,
		start.UnixMilli(),
		end.UnixMilli(),
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.airtelxstream.in/")

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

	var items []programEntry
	for _, v := range response.ProgramGuide {
		items = v
		break
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "airtelxstream.in",
			Title:       item.Title,
			Description: item.Desc,
			Lang:        channel.Lang,
			Start:       time.UnixMilli(item.StartTime),
			Stop:        time.UnixMilli(item.EndTime),
		}

		if item.Images.LandscapeHD != "" {
			p.Image = item.Images.LandscapeHD
		} else if item.Images.Landscape != "" {
			p.Image = item.Images.Landscape
		} else if item.Images.FeatureBannerHD != "" {
			p.Image = item.Images.FeatureBannerHD
		}

		if len(item.Genres) > 0 {
			p.Category = strings.Join(item.Genres, ", ")
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	ProgramGuide map[string][]programEntry `json:"programGuide"`
}

type programEntry struct {
	Title     string   `json:"title"`
	Desc      string   `json:"desc"`
	StartTime int64    `json:"startTime"`
	EndTime   int64    `json:"endTime"`
	Images    images   `json:"images"`
	Genres    []string `json:"genres"`
}

type images struct {
	LandscapeHD     string `json:"LANDSCAPE_169_HD"`
	Landscape       string `json:"LANDSCAPE_169"`
	FeatureBannerHD string `json:"FEATURE_BANNER_HD"`
	LogoHD          string `json:"LOGO_HD"`
	Logo            string `json:"LOGO"`
}
