package deltanl

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

func (p *provider) Name() string        { return "webtv.delta.nl" }
func (p *provider) Description() string { return "Delta TV Netherlands" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "webtv.delta.nl")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour).UTC()
	end := start.Add(24 * time.Hour)

	url := fmt.Sprintf(
		"https://clientapi.tv.delta.nl/guide/channels/list?start=%d&end=%d&includeDetails=true&channels=%s",
		start.Unix(),
		end.Unix(),
		channel.SiteID,
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

	programs, err := Parse(body, channel)
	if err != nil {
		return nil, err
	}

	for i, prog := range programs {
		if prog.programID == "" {
			continue
		}
		desc, err := p.fetchDetails(prog.programID)
		if err != nil {
			continue
		}
		programs[i].Description = desc
	}

	result := make([]domain.Program, len(programs))
	for i, p := range programs {
		result[i] = p.Program
	}
	return result, nil
}

func (p *provider) fetchDetails(programID string) (string, error) {
	url := fmt.Sprintf(
		"https://clientapi.tv.delta.nl/guide/4/details/%s?X-Response-Version=4.5",
		programID,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var detail detailResponse
	if err := json.Unmarshal(body, &detail); err != nil {
		return "", err
	}

	return detail.Description, nil
}

type parsedProgram struct {
	domain.Program
	programID string
}

func Parse(data []byte, channel domain.Channel) ([]parsedProgram, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, nil
	}

	guideData, ok := raw[channel.SiteID]
	if !ok {
		return nil, nil
	}

	var items []guideItem
	if err := json.Unmarshal(guideData, &items); err != nil {
		return nil, nil
	}

	programs := make([]parsedProgram, 0, len(items))
	for _, item := range items {
		p := parsedProgram{
			Program: domain.Program{
				Channel: channel.XmltvID,
				Source:  "webtv.delta.nl",
				Title:   item.Title,
				Lang:    channel.Lang,
				Start:   time.Unix(item.Start, 0).UTC(),
				Stop:    time.Unix(item.End, 0).UTC(),
			},
			programID: item.ID,
		}

		if item.Images.Thumbnail.URL != "" {
			p.Image = item.Images.Thumbnail.URL
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type guideItem struct {
	ID     string `json:"ID"`
	Title  string `json:"title"`
	Start  int64  `json:"start"`
	End    int64  `json:"end"`
	Images struct {
		Thumbnail struct {
			URL string `json:"url"`
		} `json:"thumbnail"`
	} `json:"images"`
}

type detailResponse struct {
	Description string `json:"description"`
}
