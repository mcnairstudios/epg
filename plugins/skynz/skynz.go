package skynz

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

func (p *provider) Name() string        { return "sky.co.nz" }
func (p *provider) Description() string { return "Sky NZ TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "sky.co.nz")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour).UTC()
	end := start.Add(24 * time.Hour)

	url := fmt.Sprintf(
		"https://web-epg.sky.co.nz/prod/epgs/v1?channelNumber=%s&start=%d&end=%d&limit=20000",
		channel.SiteID,
		start.UnixMilli(),
		end.UnixMilli(),
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
		return []domain.Program{}, nil
	}

	if response.Events == nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(response.Events))
	for _, item := range response.Events {
		startMs, err := strconv.ParseInt(item.Start, 10, 64)
		if err != nil {
			continue
		}
		endMs, err := strconv.ParseInt(item.End, 10, 64)
		if err != nil {
			continue
		}

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "sky.co.nz",
			Title:       item.Title,
			Description: item.Synopsis,
			Lang:        channel.Lang,
			Start:       time.UnixMilli(startMs).UTC(),
			Stop:        time.UnixMilli(endMs).UTC(),
		}

		if len(item.Genres) > 0 {
			p.Category = item.Genres[0]
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Events []event `json:"events"`
}

type event struct {
	Title    string   `json:"title"`
	Synopsis string   `json:"synopsis"`
	Start    string   `json:"start"`
	End      string   `json:"end"`
	Genres   []string `json:"genres"`
	Rating   string   `json:"rating"`
}
