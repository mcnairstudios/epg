package plextv

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

const apiEndpoint = "https://epg.provider.plex.tv"

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

func (p *provider) Name() string        { return "plex.tv" }
func (p *provider) Description() string { return "Plex TV live channels" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "plex.tv")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("%s/grid?channelGridKey=%s&date=%s", apiEndpoint, channel.SiteID, date.Format("2006-01-02"))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-plex-provider-version", "7.2")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from plextv", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response mediaContainer
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	type segment struct {
		beginsAt int64
		endsAt   int64
		meta     metadata
	}

	var segments []segment
	for _, item := range response.MediaContainer.Metadata {
		sorted := make([]media, len(item.Media))
		copy(sorted, item.Media)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].BeginsAt < sorted[j].BeginsAt })

		var merged []media
		for _, m := range sorted {
			if len(merged) > 0 && merged[len(merged)-1].EndsAt == m.BeginsAt {
				merged[len(merged)-1].EndsAt = m.EndsAt
			} else {
				merged = append(merged, m)
			}
		}

		for _, m := range merged {
			segments = append(segments, segment{beginsAt: m.BeginsAt, endsAt: m.EndsAt, meta: item})
		}
	}

	sort.Slice(segments, func(i, j int) bool { return segments[i].beginsAt < segments[j].beginsAt })

	programs := make([]domain.Program, 0, len(segments))
	for _, seg := range segments {
		item := seg.meta
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "plex.tv",
			Lang:    channel.Lang,
			Start:   time.Unix(seg.beginsAt, 0).UTC(),
			Stop:    time.Unix(seg.endsAt, 0).UTC(),
		}

		if item.GrandparentTitle != "" {
			p.Title = item.GrandparentTitle
			if item.Title != item.GrandparentTitle {
				p.SubTitle = item.Title
			}
		} else {
			p.Title = item.Title
		}

		p.Description = item.Summary
		p.Season = item.ParentIndex
		p.Episode = item.Index

		if item.Thumb != "" {
			p.Image = item.Thumb
		} else if item.GrandparentThumb != "" {
			p.Image = item.GrandparentThumb
		}

		if len(item.Genre) > 0 {
			p.Category = item.Genre[0].Tag
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type mediaContainer struct {
	MediaContainer struct {
		Metadata []metadata `json:"Metadata"`
	} `json:"MediaContainer"`
}

type metadata struct {
	Title            string  `json:"title"`
	GrandparentTitle string  `json:"grandparentTitle"`
	GrandparentThumb string  `json:"grandparentThumb"`
	Thumb            string  `json:"thumb"`
	Summary          string  `json:"summary"`
	ContentRating    string  `json:"contentRating"`
	ParentIndex      int     `json:"parentIndex"`
	Index            int     `json:"index"`
	Year             int     `json:"year"`
	Genre            []genre `json:"Genre"`
	Media            []media `json:"Media"`
	OriginallyAvailableAt string `json:"originallyAvailableAt"`
}

type genre struct {
	Tag string `json:"tag"`
}

type media struct {
	BeginsAt int64 `json:"beginsAt"`
	EndsAt   int64 `json:"endsAt"`
}
