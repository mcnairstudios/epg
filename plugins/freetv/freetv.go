package freetv

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func (p *provider) Name() string        { return "freetv.tv" }
func (p *provider) Description() string { return "FreeTV Israel TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "freetv.tv")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	loc, _ := time.LoadLocation("Asia/Jerusalem")
	localDate := date.In(loc)
	since := localDate.Truncate(24 * time.Hour).Format("2006-01-02T15:04-0700")
	till := localDate.Truncate(24*time.Hour).Add(24*time.Hour).Format("2006-01-02T15:04-0700")

	u := fmt.Sprintf(
		"https://web.freetv.tv/api/products/lives/programmes?liveId[]=%s&since=%s&till=%s&lang=HEB&platform=BROWSER",
		channel.SiteID, url.QueryEscape(since), url.QueryEscape(till),
	)

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from freetv", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var items []programItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		start, err := time.Parse(time.RFC3339, item.Since)
		if err != nil {
			continue
		}
		stop, err := time.Parse(time.RFC3339, item.Till)
		if err != nil {
			continue
		}

		prog := domain.Program{
			Channel: channel.XmltvID,
			Source:  "freetv.tv",
			Title:   item.Title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		}

		if item.Description != "" {
			prog.Description = item.Description
		} else if item.Lead != "" {
			prog.Description = item.Lead
		}

		if item.Images != nil {
			if imgs, ok := item.Images["16x9"]; ok && len(imgs) > 0 && imgs[0].URL != "" {
				prog.Image = "https:" + imgs[0].URL
			}
		}

		programs = append(programs, prog)
	}

	return programs, nil
}

type programItem struct {
	Title       string                `json:"title"`
	Lead        string                `json:"lead"`
	Description string                `json:"description"`
	Since       string                `json:"since"`
	Till        string                `json:"till"`
	Images      map[string][]imageRef `json:"images"`
}

type imageRef struct {
	URL string `json:"url"`
}
