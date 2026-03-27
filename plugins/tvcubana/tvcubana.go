package tvcubana

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

var havanaTZ, _ = time.LoadLocation("America/Havana")

var daysOfWeek = []string{"domingo", "lunes", "martes", "miercoles", "jueves", "viernes", "sabado"}

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "tvcubana.icrt.cu" }
func (p *provider) Description() string { return "TV Cubana program guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvcubana.icrt.cu")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	dayName := daysOfWeek[date.Weekday()]
	url := fmt.Sprintf("https://www.tvcubana.icrt.cu/cartv/%s/%s.php", channel.SiteID, dayName)

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
	var items []apiItem
	if err := json.Unmarshal(data, &items); err != nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		start, err := time.ParseInLocation("2006-01-02T15:04:05", item.EventInitialDateTime, havanaTZ)
		if err != nil {
			continue
		}
		stop, err := time.ParseInLocation("2006-01-02T15:04:05", item.EventEndDateTime, havanaTZ)
		if err != nil {
			continue
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tvcubana.icrt.cu",
			Title:       item.Title,
			Description: item.Description,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		})
	}

	return programs, nil
}

type apiItem struct {
	Title                string `json:"title"`
	Description          string `json:"description"`
	EventInitialDateTime string `json:"eventInitialDateTime"`
	EventEndDateTime     string `json:"eventEndDateTime"`
}
