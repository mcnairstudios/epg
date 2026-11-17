package xumotv

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

func (p *provider) Name() string        { return "xumo.tv" }
func (p *provider) Description() string { return "Xumo TV free streaming service" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "xumo.tv")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()
	parts := strings.SplitN(channel.SiteID, "#", 2)
	offset := parts[0]

	url := fmt.Sprintf(
		"https://valencia-app-mds.xumo.com/v2/epg/10006/%s/0.json?f=asset.title&f=asset.descriptions&limit=50&offset=%s",
		d.Format("20060102"),
		offset,
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
		return nil, fmt.Errorf("HTTP %d from xumotv", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	items := ParseItems(data, channel)
	if len(items) == 0 {
		return nil, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		p := domain.Program{
			Channel:  channel.XmltvID,
			Source:   "xumo.tv",
			Title:    item.Title,
			SubTitle: item.EpisodeTitle,
			Lang:     channel.Lang,
		}

		if item.Descriptions != nil {
			if item.Descriptions.Medium != "" {
				p.Description = item.Descriptions.Medium
			} else if item.Descriptions.Small != "" {
				p.Description = item.Descriptions.Small
			} else if item.Descriptions.Tiny != "" {
				p.Description = item.Descriptions.Tiny
			}
		}

		if item.Start != "" {
			if t, err := time.Parse("2006-01-02T15:04:05+0000", item.Start); err == nil {
				p.Start = t.UTC()
			} else if t, err := time.Parse(time.RFC3339, item.Start); err == nil {
				p.Start = t.UTC()
			}
		}
		if item.End != "" {
			if t, err := time.Parse("2006-01-02T15:04:05+0000", item.End); err == nil {
				p.Stop = t.UTC()
			} else if t, err := time.Parse(time.RFC3339, item.End); err == nil {
				p.Stop = t.UTC()
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func ParseItems(data []byte, channel domain.Channel) []MergedItem {
	if len(data) == 0 {
		return nil
	}

	var response EPGResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil
	}

	parts := strings.SplitN(channel.SiteID, "#", 2)
	if len(parts) < 2 {
		return nil
	}
	channelID := parts[1]

	var channelData *ChannelSchedule
	for i := range response.Channels {
		if fmt.Sprintf("%d", response.Channels[i].ChannelID) == channelID {
			channelData = &response.Channels[i]
			break
		}
	}

	if channelData == nil {
		return nil
	}

	items := make([]MergedItem, 0, len(channelData.Schedule))
	for _, s := range channelData.Schedule {
		asset, ok := response.Assets[s.AssetID]
		if !ok {
			continue
		}
		items = append(items, MergedItem{
			AssetID:      s.AssetID,
			Start:        s.Start,
			End:          s.End,
			Title:        asset.Title,
			EpisodeTitle: asset.EpisodeTitle,
			Descriptions: asset.Descriptions,
		})
	}

	return items
}

type EPGResponse struct {
	Channels []ChannelSchedule  `json:"channels"`
	Assets   map[string]Asset   `json:"assets"`
}

type ChannelSchedule struct {
	ChannelID int              `json:"channelId"`
	Schedule  []ScheduleEntry  `json:"schedule"`
}

type ScheduleEntry struct {
	AssetID string `json:"assetId"`
	Start   string `json:"start"`
	End     string `json:"end"`
}

type Asset struct {
	Title        string        `json:"title"`
	EpisodeTitle string        `json:"episodeTitle"`
	Descriptions *Descriptions `json:"descriptions"`
}

type Descriptions struct {
	Tiny   string `json:"tiny"`
	Small  string `json:"small"`
	Medium string `json:"medium"`
}

type MergedItem struct {
	AssetID      string
	Start        string
	End          string
	Title        string
	EpisodeTitle string
	Descriptions *Descriptions
}

func (p *provider) Timezone() string { return "America/New_York" }
