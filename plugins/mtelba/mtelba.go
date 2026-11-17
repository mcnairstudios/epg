package mtelba

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpcache"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{
		client: httpcache.Wrap(httpclient.Browser()),
	})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "mtel.ba" }
func (p *provider) Description() string { return "Mtel Bosnia TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "mtel.ba")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	parts := strings.SplitN(channel.SiteID, "#", 2)
	platform := parts[0]
	d := date.Format("2006-01-02")

	url := fmt.Sprintf(
		"https://mtel.ba/hybris/ecommerce/b2c/v1/products/channels/epg?platform=tv-%s&pageSize=999&date=%s",
		platform, d,
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
		return nil, fmt.Errorf("HTTP %d from mtelba", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

var sarajevo *time.Location


func init() {
	sarajevo, _ = time.LoadLocation("Europe/Sarajevo")
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return []domain.Program{}, nil
	}

	if response.Products == nil {
		return []domain.Program{}, nil
	}

	parts := strings.SplitN(channel.SiteID, "#", 2)
	if len(parts) < 2 {
		return []domain.Program{}, nil
	}
	channelID := parts[1]

	var channelData *product
	for i := range response.Products {
		if response.Products[i].Code == channelID {
			channelData = &response.Products[i]
			break
		}
	}

	if channelData == nil || channelData.Programs == nil {
		return []domain.Program{}, nil
	}

	var filtered []programItem
	for _, p := range channelData.Programs {
		if strings.Contains(p.Title, "Nema informacija o programu") {
			continue
		}
		filtered = append(filtered, p)
	}

	sort.Slice(filtered, func(i, j int) bool {
		ti := parseTime(filtered[i].Start)
		tj := parseTime(filtered[j].Start)
		return ti.Before(tj)
	})

	programs := make([]domain.Program, 0, len(filtered))
	for _, item := range filtered {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "mtel.ba",
			Title:       item.Title,
			Description: item.Description,
			Lang:        channel.Lang,
			Start:       parseTime(item.Start).UTC(),
			Stop:        parseTime(item.End).UTC(),
		}

		if item.Category != "" {
			p.Category = item.Category
		}

		if item.Picture != nil && item.Picture.URL != "" {
			p.Image = item.Picture.URL
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func parseTime(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, sarajevo)
	if err != nil {
		return time.Time{}
	}
	return t
}

type apiResponse struct {
	Products []product `json:"products"`
}

type product struct {
	Code     string        `json:"code"`
	Programs []programItem `json:"programs"`
}

type programItem struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Start       string   `json:"start"`
	End         string   `json:"end"`
	Picture     *picture `json:"picture"`
}

type picture struct {
	URL string `json:"url"`
}

func (p *provider) Timezone() string { return "Europe/Sarajevo" }
