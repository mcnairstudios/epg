package sfrfr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func (p *provider) Name() string        { return "tv.sfr.fr" }
func (p *provider) Description() string { return "SFR TV France" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tv.sfr.fr")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://static-cdn.tv.sfr.net/data/epg/gen8/guide_web_%s.json", date.Format("20060102"))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from sfrfr", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, nil
	}

	epgRaw, ok := raw["epg"]
	if !ok {
		return nil, nil
	}

	var epg map[string][]programItem
	if err := json.Unmarshal(epgRaw, &epg); err != nil {
		return nil, nil
	}

	items, ok := epg[channel.SiteID]
	if !ok {
		return nil, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tv.sfr.fr",
			Title:       item.Title,
			SubTitle:    item.SubTitle,
			Description: item.LongSynopsis,
			Category:    item.Genre,
			Season:      item.SeasonNumber,
			Episode:     item.EpisodeNumber,
			Lang:        channel.Lang,
		}

		if item.StartDate > 0 {
			p.Start = time.UnixMilli(item.StartDate).UTC()
		}
		if item.EndDate > 0 {
			p.Stop = time.UnixMilli(item.EndDate).UTC()
		}

		if len(item.Images) > 0 {
			for _, img := range item.Images {
				if img.Type == "landscape" {
					p.Image = img.URL
					break
				}
			}
			if p.Image == "" {
				p.Image = item.Images[0].URL
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type programItem struct {
	Title         string      `json:"title"`
	SubTitle      string      `json:"subTitle"`
	Genre         string      `json:"genre"`
	LongSynopsis  string      `json:"longSynopsis"`
	StartDate     int64       `json:"startDate"`
	EndDate       int64       `json:"endDate"`
	SeasonNumber  int         `json:"seasonNumber"`
	EpisodeNumber int         `json:"episodeNumber"`
	Images        []imageItem `json:"images"`
}

type imageItem struct {
	URL  string `json:"url"`
	Type string `json:"type"`
}

func (p *provider) Timezone() string { return "Europe/Paris" }
