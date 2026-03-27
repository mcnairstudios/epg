package horizontv

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func (p *provider) Name() string        { return "horizon.tv" }
func (p *provider) Description() string { return "Horizon TV Slovakia EPG" }

const apiEndpoint = "https://legacy-static.oesp.horizon.tv/oesp/v4/SK/slk/web"

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "horizontv-10024", Name: "AMC", Source: "horizon.tv", SiteID: "10024", Lang: "sk", XmltvID: "AMCCzechRepublic.cz"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Format("20060102")
	var allItems []scheduleItem
	for page := 1; page <= 4; page++ {
		url := fmt.Sprintf("%s/programschedules/%s/%d", apiEndpoint, d, page)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		resp, err := p.client.Do(req)
		if err != nil {
			if page == 1 {
				return nil, err
			}
			continue
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		items := extractItems(body, channel.SiteID)
		allItems = append(allItems, items...)
		if page == 1 && len(items) == 0 {
			return []domain.Program{}, nil
		}
	}

	programs := make([]domain.Program, 0, len(allItems))
	for _, item := range allItems {
		prog := domain.Program{
			Channel: channel.XmltvID,
			Source:  "horizon.tv",
			Title:   item.Title,
			Lang:    channel.Lang,
			Start:   time.UnixMilli(item.Start).UTC(),
			Stop:    time.UnixMilli(item.End).UTC(),
		}

		if item.ListingID != "" {
			detail := p.fetchDetail(item.ListingID)
			if detail != nil && detail.Program != nil {
				prog.Description = detail.Program.LongDescription
				prog.Season = parseSeason(detail.Program.SeriesNumber)
				prog.Episode = parseEpisode(detail.Program.SeriesEpisodeNumber)
			}
		}

		programs = append(programs, prog)
	}

	return programs, nil
}

func (p *provider) fetchDetail(listingID string) *listingDetail {
	url := fmt.Sprintf("%s/listings/%s", apiEndpoint, listingID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	var detail listingDetail
	if err := json.Unmarshal(body, &detail); err != nil {
		return nil
	}
	return &detail
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	items := extractItems(data, channel.SiteID)
	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "horizon.tv",
			Title:   item.Title,
			Lang:    channel.Lang,
			Start:   time.UnixMilli(item.Start).UTC(),
			Stop:    time.UnixMilli(item.End).UTC(),
		})
	}
	return programs, nil
}

func ParseWithDetails(scheduleSegments [][]byte, details map[string][]byte, channel domain.Channel) ([]domain.Program, error) {
	var allItems []scheduleItem
	for _, seg := range scheduleSegments {
		allItems = append(allItems, extractItems(seg, channel.SiteID)...)
	}

	programs := make([]domain.Program, 0, len(allItems))
	for _, item := range allItems {
		prog := domain.Program{
			Channel: channel.XmltvID,
			Source:  "horizon.tv",
			Title:   item.Title,
			Lang:    channel.Lang,
			Start:   time.UnixMilli(item.Start).UTC(),
			Stop:    time.UnixMilli(item.End).UTC(),
		}

		if item.ListingID != "" {
			if detailData, ok := details[item.ListingID]; ok {
				var detail listingDetail
				if err := json.Unmarshal(detailData, &detail); err == nil && detail.Program != nil {
					prog.Description = detail.Program.LongDescription
					prog.Season = parseSeason(detail.Program.SeriesNumber)
					prog.Episode = parseEpisode(detail.Program.SeriesEpisodeNumber)
				}
			}
		}

		programs = append(programs, prog)
	}
	return programs, nil
}

func extractItems(data []byte, siteID string) []scheduleItem {
	var resp scheduleResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil
	}
	prefix := "lgi-obolite-sk-prod-master:" + siteID
	for _, entry := range resp.Entries {
		if entry.Owner == prefix {
			return entry.Listings
		}
	}
	return nil
}

func parseSeason(v json.Number) int {
	if v == "" {
		return 0
	}
	n, err := v.Int64()
	if err != nil {
		return 0
	}
	s := v.String()
	if len(s) > 2 {
		return 0
	}
	return int(n)
}

func parseEpisode(v json.Number) int {
	if v == "" {
		return 0
	}
	n, err := v.Int64()
	if err != nil {
		return 0
	}
	s := v.String()
	if len(s) > 3 {
		return 0
	}
	return int(n)
}

type scheduleResponse struct {
	Entries []scheduleEntry `json:"entries"`
}

type scheduleEntry struct {
	Owner    string         `json:"o"`
	Listings []scheduleItem `json:"l"`
}

type scheduleItem struct {
	ListingID string `json:"i"`
	Title     string `json:"t"`
	Start     int64  `json:"s"`
	End       int64  `json:"e"`
}

type listingDetail struct {
	Program *programDetail `json:"program"`
}

type programDetail struct {
	LongDescription     string      `json:"longDescription"`
	SeriesNumber        json.Number `json:"seriesNumber"`
	SeriesEpisodeNumber json.Number `json:"seriesEpisodeNumber"`
	Year                string      `json:"year"`
	Categories          []category  `json:"categories"`
	Cast                []string    `json:"cast"`
	Directors           []string    `json:"directors"`
}

type category struct {
	Title string `json:"title"`
}
