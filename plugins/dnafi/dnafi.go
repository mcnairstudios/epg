package dnafi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func (p *provider) Name() string        { return "dna.fi" }
func (p *provider) Description() string { return "DNA Finland TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "dna.fi")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()
	beginTS := d.Add(2 * time.Hour).UnixMilli()
	endTS := d.Add(26*time.Hour - time.Second).UnixMilli()

	url := fmt.Sprintf(
		"https://mts-pro-envoy-vip.dna.fi/hbx/api/pub/xrtv/g/media?q=channel:%s&q=profile:pr&q=start-interval:%d/%d",
		channel.SiteID, beginTS, endTS,
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return []domain.Program{}, nil
	}

	items := resp.Embedded.MediaItems
	if items == nil {
		return []domain.Program{}, nil
	}

	d := date.Truncate(24 * time.Hour).UTC()
	programs := make([]domain.Program, 0)

	for _, item := range items {
		meta := item.Embedded.Meta.Data
		if meta.Start == 0 {
			continue
		}

		startTime := time.UnixMilli(meta.Start).UTC()
		if startTime.Year() != d.Year() || startTime.YearDay() != d.YearDay() {
			continue
		}

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "dna.fi",
			Title:   meta.Title.Value,
			SubTitle: meta.EpisodeTitle.Value,
			Description: meta.Description.Value,
			Season:  meta.SeasonNumber,
			Episode: meta.EpisodeNumber,
			Lang:    channel.Lang,
			Start:   startTime,
			Stop:    time.UnixMilli(meta.End).UTC(),
		}

		cats := item.Embedded.MediaCategories
		if len(cats) > 0 {
			p.Category = cats[0].Name
		}

		images := item.Embedded.Images
		if len(images) > 0 {
			p.Image = images[0].Src
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Embedded struct {
		MediaItems []mediaItem `json:"xrtv:media-item"`
	} `json:"_embedded"`
}

type mediaItem struct {
	Embedded struct {
		Meta            metaWrapper       `json:"xrtv:meta"`
		MediaCategories []mediaCategory   `json:"xrtv:media-category"`
		Images          []imageEntry      `json:"xrtv:image"`
	} `json:"_embedded"`
}

type metaWrapper struct {
	Data metaData `json:"data"`
}

type metaData struct {
	Title         langValue `json:"title"`
	EpisodeTitle  langValue `json:"episode_title"`
	Description   langValue `json:"description"`
	Start         int64     `json:"start"`
	End           int64     `json:"end"`
	SeasonNumber  int       `json:"season_number"`
	EpisodeNumber int       `json:"episode_number"`
	Year          string    `json:"year"`
}

type langValue struct {
	Lang  string `json:"lang"`
	Value string `json:"value"`
}

type mediaCategory struct {
	Name string `json:"name"`
}

type imageEntry struct {
	Src string `json:"src"`
}
