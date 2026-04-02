package tvguide

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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

func (p *provider) Name() string        { return "tvguide.com" }
func (p *provider) Description() string { return "TV Guide US television listings" }

const providerID = "9100001138"

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvguide.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour).UTC()
	url := fmt.Sprintf(
		"https://backend.tvguide.com/tvschedules/tvguide/%s/web?start=%d&duration=240",
		providerID, start.Unix(),
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", "https://www.tvguide.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from tvguide", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	schedules := extractSchedules(body, channel.SiteID)
	return fetchDetails(p.client, schedules, channel)
}

func Parse(data []byte, channel domain.Channel, detailsFetcher func(url string) *ItemDetails) ([]domain.Program, error) {
	schedules := extractSchedules(data, channel.SiteID)
	var programs []domain.Program

	for _, s := range schedules {
		prog := domain.Program{
			Channel: channel.XmltvID,
			Source:  "tvguide.com",
			Title:   s.Title,
			Lang:        channel.Lang,
			Start:   time.Unix(s.StartTime, 0).UTC(),
			Stop:    time.Unix(s.EndTime, 0).UTC(),
		}

		if detailsFetcher != nil {
			if details := detailsFetcher(s.ProgramDetails); details != nil {
				if details.Title != "" {
					prog.Title = details.Title
				}
				if details.EpisodeNumber > 0 && details.EpisodeTitle != "" {
					prog.SubTitle = details.EpisodeTitle
				}
				prog.Description = details.Description
				prog.Season = details.SeasonNumber
				prog.Episode = details.EpisodeNumber
				if details.Rating != "" {
					prog.Category = details.Rating
				}
				if len(details.Genres) > 0 {
					names := make([]string, len(details.Genres))
					for i, g := range details.Genres {
						names[i] = g.Name
					}
					prog.Category = strings.Join(names, ", ")
				}
			}
		}

		programs = append(programs, prog)
	}

	return programs, nil
}

func extractSchedules(data []byte, siteID string) []programSchedule {
	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil
	}

	if resp.Data == nil || resp.Data.Items == nil {
		return nil
	}

	var schedules []programSchedule
	for _, item := range resp.Data.Items {
		sid := formatSourceID(item.Channel.SourceID)
		if sid != siteID {
			continue
		}
		schedules = append(schedules, item.ProgramSchedules...)
	}
	return schedules
}

func formatSourceID(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var n int64
	if err := json.Unmarshal(raw, &n); err == nil {
		return strconv.FormatInt(n, 10)
	}
	return ""
}

func fetchDetails(client httpClient, schedules []programSchedule, channel domain.Channel) ([]domain.Program, error) {
	var programs []domain.Program
	for _, s := range schedules {
		prog := domain.Program{
			Channel: channel.XmltvID,
			Source:  "tvguide.com",
			Title:   s.Title,
			Lang:        channel.Lang,
			Start:   time.Unix(s.StartTime, 0).UTC(),
			Stop:    time.Unix(s.EndTime, 0).UTC(),
		}

		if s.ProgramDetails != "" {
			req, err := http.NewRequest(http.MethodGet, s.ProgramDetails, nil)
			if err == nil {
				req.Header.Set("Referer", "https://www.tvguide.com/")
				if resp, err := client.Do(req); err == nil {
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					var detResp detailsResponse
					if json.Unmarshal(body, &detResp) == nil && detResp.Data != nil && detResp.Data.Item != nil {
						d := detResp.Data.Item
						if d.Title != "" {
							prog.Title = d.Title
						}
						if d.EpisodeNumber > 0 && d.EpisodeTitle != "" {
							prog.SubTitle = d.EpisodeTitle
						}
						prog.Description = d.Description
						prog.Season = d.SeasonNumber
						prog.Episode = d.EpisodeNumber
						if len(d.Genres) > 0 {
							names := make([]string, len(d.Genres))
							for i, g := range d.Genres {
								names[i] = g.Name
							}
							prog.Category = strings.Join(names, ", ")
						}
					}
				}
			}
		}

		programs = append(programs, prog)
	}
	return programs, nil
}

type apiResponse struct {
	Data *apiData `json:"data"`
}

type apiData struct {
	Items []channelItem `json:"items"`
}

type channelItem struct {
	Channel          channelInfo       `json:"channel"`
	ProgramSchedules []programSchedule `json:"programSchedules"`
}

type channelInfo struct {
	FullName string          `json:"fullName"`
	SourceID json.RawMessage `json:"sourceId"`
}

type programSchedule struct {
	StartTime      int64  `json:"startTime"`
	EndTime        int64  `json:"endTime"`
	Title          string `json:"title"`
	ProgramDetails string `json:"programDetails"`
}

type detailsResponse struct {
	Data *detailsData `json:"data"`
}

type detailsData struct {
	Item *itemDetails `json:"item"`
}

type ItemDetails = itemDetails

type itemDetails struct {
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	EpisodeTitle  string  `json:"episodeTitle"`
	SeasonNumber  int     `json:"seasonNumber"`
	EpisodeNumber int     `json:"episodeNumber"`
	Rating        string  `json:"rating"`
	Genres        []genre `json:"genres"`
}

type genre struct {
	Name string `json:"name"`
}

func (p *provider) Timezone() string { return "America/New_York" }
