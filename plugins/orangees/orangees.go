package orangees

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "orangetv.orange.es" }
func (p *provider) Description() string { return "Orange TV Spain EPG" }

const apiProgramEndpoint = "https://epg.orangetv.orange.es/epg/Smartphone_Android/1_PRO"
const apiImageEndpoint = "https://pc.orangetv.orange.es/pc/api/rtv/v1/images"

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "orangees-1010", Name: "La 1", Source: "orangetv.orange.es", SiteID: "1010", Lang: "es", XmltvID: "La1.es"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	var allItems []programItem
	for segment := 1; segment <= 3; segment++ {
		url := fmt.Sprintf("%s/%s_8h_%d.json", apiProgramEndpoint, date.Format("20060102"), segment)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")

		resp, err := p.client.Do(req)
		if err != nil {
			if segment == 1 {
				return nil, err
			}
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		items := parseItems(body, channel.SiteID)
		allItems = append(allItems, items...)
		if segment == 1 && len(items) == 0 {
			return []domain.Program{}, nil
		}
	}

	return buildPrograms(allItems, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	items := parseItems(data, channel.SiteID)
	return buildPrograms(items, channel)
}

func ParseMulti(segments [][]byte, channel domain.Channel) ([]domain.Program, error) {
	var allItems []programItem
	for _, seg := range segments {
		allItems = append(allItems, parseItems(seg, channel.SiteID)...)
	}
	return buildPrograms(allItems, channel)
}

func buildPrograms(items []programItem, channel domain.Channel) ([]domain.Program, error) {
	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "orangetv.orange.es",
			Title:       item.Name,
			SubTitle:    item.SeriesName,
			Description: item.Description,
			Lang:        channel.Lang,
			Start:       time.UnixMilli(item.StartDate).UTC(),
			Stop:        time.UnixMilli(item.EndDate).UTC(),
		}

		if item.SeriesSeason != "" {
			if v, err := strconv.Atoi(item.SeriesSeason); err == nil {
				p.Season = v
			}
		}
		if item.EpisodeID != "" {
			if v, err := strconv.Atoi(item.EpisodeID); err == nil {
				p.Episode = v
			}
		}

		if len(item.Genres) > 0 {
			names := make([]string, 0, len(item.Genres))
			for _, g := range item.Genres {
				names = append(names, g.Name)
			}
			p.Category = strings.Join(names, ", ")
		}

		for _, att := range item.Attachments {
			if strings.EqualFold(att.Name, "cover") {
				p.Image = apiImageEndpoint + att.Value
				break
			}
		}

		programs = append(programs, p)
	}
	return programs, nil
}

func parseItems(data []byte, siteID string) []programItem {
	var channels []channelData
	if err := json.Unmarshal(data, &channels); err != nil {
		return nil
	}
	var result []programItem
	for _, ch := range channels {
		if ch.ChannelExternalID == siteID {
			result = append(result, ch.Programs...)
		}
	}
	return result
}

type channelData struct {
	ChannelExternalID string        `json:"channelExternalId"`
	Programs          []programItem `json:"programs"`
}

type programItem struct {
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	SeriesName   string       `json:"seriesName"`
	SeriesSeason string       `json:"seriesSeason"`
	EpisodeID    string       `json:"episodeId"`
	StartDate    int64        `json:"startDate"`
	EndDate      int64        `json:"endDate"`
	Genres       []genre      `json:"genres"`
	Attachments  []attachment `json:"attachments"`
}

type genre struct {
	Name string `json:"name"`
}

type attachment struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
