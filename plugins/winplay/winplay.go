package winplay

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

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

func (p *provider) Name() string        { return "winplay.co" }
func (p *provider) Description() string { return "Win+ Colombia streaming" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "winplay-futbol", Name: "Win+ Futbol", Source: "winplay.co", SiteID: "529cff6f6bd2ea6b610000e0", Lang: "es", XmltvID: "WinFutbol.co"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	req, err := http.NewRequest(http.MethodPost, "https://next.platform.mediastre.am/graphql", nil)
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
	var response graphQLResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	if response.Data == nil {
		return []domain.Program{}, nil
	}

	var schedules []scheduleItem
	for _, live := range response.Data.GetLives {
		if live.ID == channel.SiteID {
			schedules = live.Schedules
			break
		}
	}

	programs := make([]domain.Program, 0, len(schedules))
	for _, s := range schedules {
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "winplay.co",
			Title:   s.Name,
			Lang:    channel.Lang,
		}

		if t, err := time.Parse(time.RFC3339Nano, s.DateStart); err == nil {
			p.Start = t.UTC()
		}
		if t, err := time.Parse(time.RFC3339Nano, s.DateEnd); err == nil {
			p.Stop = t.UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type graphQLResponse struct {
	Data *responseData `json:"data"`
}

type responseData struct {
	GetLives []liveChannel `json:"getLives"`
}

type liveChannel struct {
	ID        string         `json:"_id"`
	Name      string         `json:"name"`
	Schedules []scheduleItem `json:"schedules"`
}

type scheduleItem struct {
	Name      string `json:"name"`
	DateStart string `json:"date_start"`
	DateEnd   string `json:"date_end"`
}
