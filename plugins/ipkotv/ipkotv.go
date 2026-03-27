package ipkotv

import (
	"bytes"
	"encoding/json"
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

func (p *provider) Name() string        { return "ipko.tv" }
func (p *provider) Description() string { return "IPKO TV Kosovo" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "ipko.tv")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour)
	end := start.AddDate(0, 0, 1)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"ch_ext_id": channel.SiteID,
		"from":      start.Unix(),
		"to":        end.Unix(),
	})

	req, err := http.NewRequest(http.MethodPost, "https://stargate.ipko.tv/api/titan.tv.WebEpg/GetWebEpgData", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://ipko.tv")
	req.Header.Set("x-language", "sq")

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

	programs := make([]domain.Program, 0, len(response.Shows))
	for _, show := range response.Shows {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "ipko.tv",
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
