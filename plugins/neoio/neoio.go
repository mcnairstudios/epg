package neoio

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
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "neo.io" }
func (p *provider) Description() string { return "NEO.io Slovenian TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "neo.io")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := "https://stargate.telekom.si/api/titan.tv.WebEpg/GetWebEpgData"

	start := date.Truncate(24 * time.Hour).UTC()
	end := start.Add(24 * time.Hour)

	body := fmt.Sprintf(`{"ch_ext_id":%q,"from":%d,"to":%d}`, channel.SiteID, start.Unix(), end.Unix())

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	_ = body

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	programs := make([]domain.Program, 0, len(response.Shows))
	for _, show := range response.Shows {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "neo.io",
			Title:       show.Title,
			Description: show.Summary,
			Image:       show.Thumbnail,
			Lang:        channel.Lang,
			Start:       time.Unix(show.ShowStart, 0).UTC(),
			Stop:        time.Unix(show.ShowEnd, 0).UTC(),
		}
		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Shows []show `json:"shows"`
}

type show struct {
	Title     string `json:"title"`
	ShowStart int64  `json:"show_start"`
	ShowEnd   int64  `json:"show_end"`
	Summary   string `json:"summary"`
	Thumbnail string `json:"thumbnail"`
}
