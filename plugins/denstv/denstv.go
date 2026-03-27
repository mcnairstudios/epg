package denstv

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

var jakartaTZ = time.FixedZone("Asia/Jakarta", 7*60*60)

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

func (p *provider) Name() string        { return "dens.tv" }
func (p *provider) Description() string { return "Dens.tv Indonesian streaming" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "denstv-1", Name: "Dens TV", Source: "dens.tv", SiteID: "1", Lang: "id", XmltvID: "DensTV.id"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://www.dens.tv/api/dens3/tv/TvChannels/listEpgByDate?date=%s&id_channel=%s&app_type=10",
		date.Format("2006-01-02"),
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

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	programs := make([]domain.Program, 0, len(response.Data))
	for _, item := range response.Data {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "dens.tv",
			Title:       item.Title,
			Description: item.Description,
			Lang:        channel.Lang,
		}

		if t, err := time.ParseInLocation("2006-01-02 15:04:05", item.StartTime, jakartaTZ); err == nil {
			p.Start = t.UTC()
		}
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", item.EndTime, jakartaTZ); err == nil {
			p.Stop = t.UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Data []epgItem `json:"data"`
}

type epgItem struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
}
