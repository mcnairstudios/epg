package abcau

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

func (p *provider) Name() string        { return "abc.net.au" }
func (p *provider) Description() string { return "Australian Broadcasting Corporation" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "abc.net.au")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	region := strings.Split(channel.SiteID, "#")[0]
	url := fmt.Sprintf("https://cdn.iview.abc.net.au/epg/processed/%s_%s.json", region, date.Format("2006-01-02"))

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
		return nil, fmt.Errorf("HTTP %d from abcau", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

var sydneyTZ = mustLoadLocation("Australia/Sydney")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response scheduleResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil
	}

	parts := strings.Split(channel.SiteID, "#")
	if len(parts) < 2 {
		return nil, nil
	}
	channelID := parts[1]

	var listings []listing
	for _, s := range response.Schedule {
		if s.Channel == channelID {
			listings = s.Listing
			break
		}
	}

	programs := make([]domain.Program, 0, len(listings))
	for _, item := range listings {
		start := parseTime(item.StartTime)
		stop := parseTime(item.EndTime)

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "abc.net.au",
			Title:       item.Title,
			SubTitle:    item.EpisodeTitle,
			Description: item.Description,
			Season:      item.SeriesNum,
			Episode:     item.EpisodeNum,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		}

		if item.ImageFile != "" {
			p.Image = "https://www.abc.net.au/tv/common/images/publicity/" + item.ImageFile
		}

		if len(item.Genres) > 0 {
			p.Category = item.Genres[0]
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func parseTime(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02T15:04:05", s, sydneyTZ)
	if err != nil {
		return time.Time{}
	}
	return t
}

type scheduleResponse struct {
	Schedule []scheduleChannel `json:"schedule"`
}

type scheduleChannel struct {
	Channel string    `json:"channel"`
	Listing []listing `json:"listing"`
}

type listing struct {
	StartTime    string   `json:"start_time"`
	EndTime      string   `json:"end_time"`
	Title        string   `json:"title"`
	EpisodeTitle string   `json:"episode_title"`
	Description  string   `json:"description"`
	Rating       string   `json:"rating"`
	Genres       []string `json:"genres"`
	ImageFile    string   `json:"image_file"`
	SeriesNum    int      `json:"series_num"`
	EpisodeNum   int      `json:"episode_num"`
}

func (p *provider) Timezone() string { return "Australia/Sydney" }
