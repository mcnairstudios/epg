package rtsch

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

func (p *provider) Name() string        { return "rts.ch" }
func (p *provider) Description() string { return "RTS Swiss public television" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "rts.ch")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.UTC().Format("2006-01-02")
	url := fmt.Sprintf(
		"https://il.srgssr.ch/integrationlayer/2.0/rts/programGuide/tv/byDate/%s?reduced=false&channelId=%s",
		d, channel.SiteID,
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
		return nil, fmt.Errorf("HTTP %d from rtsch", resp.StatusCode)
	}

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

	if len(response.ProgramGuide) == 0 || response.ProgramGuide[0].ProgramList == nil {
		return nil, nil
	}

	list := response.ProgramGuide[0].ProgramList
	programs := make([]domain.Program, 0, len(list))
	for _, item := range list {
		start, err := time.Parse(time.RFC3339, item.StartTime)
		if err != nil {
			continue
		}
		stop, err := time.Parse(time.RFC3339, item.EndTime)
		if err != nil {
			continue
		}

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "rts.ch",
			Title:   item.Title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		}

		if item.Subtitle != "" {
			p.SubTitle = item.Subtitle
		}
		if item.Description != "" {
			p.Description = item.Description
		} else if item.Show != nil && item.Show.Description != "" {
			p.Description = item.Show.Description
		}
		if item.Genre != "" {
			p.Category = item.Genre
		}
		if item.ImageURL != "" {
			p.Image = item.ImageURL
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	ProgramGuide []programGuide `json:"programGuide"`
}

type programGuide struct {
	ProgramList []programItem `json:"programList"`
}

type programItem struct {
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle"`
	Description string `json:"description"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	ImageURL    string `json:"imageUrl"`
	Genre       string `json:"genre"`
	Show        *show  `json:"show"`
}

type show struct {
	Description string `json:"description"`
}

func (p *provider) Timezone() string { return "Europe/Zurich" }
