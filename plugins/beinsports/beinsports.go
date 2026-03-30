package beinsports

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
	plugin.Register(&provider{client: httpclient.Browser()})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "beinsports.com" }
func (p *provider) Description() string { return "beIN Sports EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "beinsports.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour).UTC()
	end := start.Add(24 * time.Hour)

	url := fmt.Sprintf(
		"https://www.beinsports.com/api/opta/tv-event?&startBefore=%sZ&endAfter=%sZ&channelIds=%s",
		end.Format("2006-01-02T15:04:05.000"),
		start.Format("2006-01-02T15:04:05.000"),
		channel.SiteID,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from beinsports", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return []domain.Program{}, nil
	}

	if len(resp.Rows) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(resp.Rows))
	for _, item := range resp.Rows {
		start, err := time.Parse(time.RFC3339Nano, item.StartDate)
		if err != nil {
			start, err = time.Parse("2006-01-02T15:04:05.000Z", item.StartDate)
			if err != nil {
				continue
			}
		}

		stop, err := time.Parse(time.RFC3339Nano, item.EndDate)
		if err != nil {
			stop, err = time.Parse("2006-01-02T15:04:05.000Z", item.EndDate)
			if err != nil {
				continue
			}
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "beinsports.com",
			Title:       item.Title,
			Description: item.Description,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

type apiResponse struct {
	Count int       `json:"count"`
	Rows  []apiItem `json:"rows"`
}

type apiItem struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
}
