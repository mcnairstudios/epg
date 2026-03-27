package abcau

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func (p *provider) Name() string        { return "abc.net.au" }
func (p *provider) Description() string { return "ABC Australia TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "abcau-ABC1-Sydney", Name: "ABC1", Source: "abc.net.au", SiteID: "Sydney#ABC1", Lang: "en", XmltvID: "ABC1.au"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	region := strings.SplitN(channel.SiteID, "#", 2)[0]
	url := fmt.Sprintf(
		"https://cdn.iview.abc.net.au/epg/processed/%s_%s.json",
		region,
		date.Format("2006-01-02"),
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

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil
	}

	channelID := ""
	parts := strings.SplitN(channel.SiteID, "#", 2)
	if len(parts) == 2 {
		channelID = parts[1]
	}

	var listings []listing
	for _, s := range response.Schedule {
		if s.Channel == channelID {
			listings = s.Listing
			break
		}
	}

	loc, _ := time.LoadLocation("Australia/Sydney")
	programs := make([]domain.Program, 0, len(listings))
	for _, item := range listings {
		prog := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "abc.net.au",
			Title:       item.Title,
			SubTitle:    item.EpisodeTitle,
			Description: item.Description,
			Lang:        channel.Lang,
		}

		if item.SeriesNum > 0 {
			prog.Season = item.SeriesNum
		}
		if item.EpisodeNum > 0 {
			prog.Episode = item.EpisodeNum
		}

		if item.ImageFile != "" {
			prog.Image = "https://www.abc.net.au/tv/common/images/publicity/" + item.ImageFile
		}

		if item.StartTime != "" {
			if t, err := time.ParseInLocation("2006-01-02T15:04:05", item.StartTime, loc); err == nil {
				prog.Start = t
			}
		}
		if item.EndTime != "" {
			if t, err := time.ParseInLocation("2006-01-02T15:04:05", item.EndTime, loc); err == nil {
				prog.Stop = t
			}
		}

		programs = append(programs, prog)
	}

	return programs, nil
}

type apiResponse struct {
	Schedule []schedule `json:"schedule"`
}

type schedule struct {
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
	SeriesNum    int      `json:"series_num"`
	EpisodeNum   int      `json:"episode_num"`
	ImageFile    string   `json:"image_file"`
	Genres       []string `json:"genres"`
}
