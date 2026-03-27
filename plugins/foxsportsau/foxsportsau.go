package foxsportsau

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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

func (p *provider) Name() string        { return "foxsports.com.au" }
func (p *provider) Description() string { return "Fox Sports Australia TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "foxsports.com.au")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour).UTC()
	end := start.Add(24 * time.Hour)

	url := fmt.Sprintf(
		"https://tvguide.foxsports.com.au/granite-api/programmes.json?from=%s&to=%s",
		start.Format("2006-01-02"),
		end.Format("2006-01-02"),
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
		return nil, err
	}

	programs := make([]domain.Program, 0)
	for _, item := range response.ChannelProgramme {
		if strconv.Itoa(item.ChannelID) != channel.SiteID {
			continue
		}

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "foxsports.com.au",
			Title:       item.ProgrammeTitle,
			SubTitle:    item.Title,
			Description: item.Synopsis,
			Category:    item.GenreTitle,
			Lang:        channel.Lang,
		}

		if item.StartTime != "" {
			if t, err := time.Parse(time.RFC3339, item.StartTime); err == nil {
				p.Start = t.UTC()
			}
		}
		if item.EndTime != "" {
			if t, err := time.Parse(time.RFC3339, item.EndTime); err == nil {
				p.Stop = t.UTC()
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	ChannelProgramme []programme `json:"channel-programme"`
}

type programme struct {
	ProgrammeTitle string `json:"programmeTitle"`
	Title          string `json:"title"`
	StartTime      string `json:"startTime"`
	EndTime        string `json:"endTime"`
	ChannelID      int    `json:"channelId"`
	ChannelName    string `json:"channelName"`
	GenreTitle     string `json:"genreTitle"`
	Synopsis       string `json:"synopsis"`
}
