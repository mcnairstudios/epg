package gigatv

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

var bangkok *time.Location

func init() {
	var err error
	bangkok, err = time.LoadLocation("Asia/Bangkok")
	if err != nil {
		panic(err)
	}

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

func (p *provider) Name() string        { return "gigatv.3bbtv.co.th" }
func (p *provider) Description() string { return "GigaTV 3BB Thailand EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "gigatv.3bbtv.co.th")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://gigatv.3bbtv.co.th/wp-content/themes/changwattana/epg/%s.json",
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from gigatv", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	var items []apiItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, nil
	}

	dateUTC := date.UTC()
	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		start, err := time.ParseInLocation("2006-01-02 15:04:05", item.StartTime, bangkok)
		if err != nil {
			continue
		}

		startUTC := start.UTC()
		if startUTC.Year() != dateUTC.Year() || startUTC.YearDay() != dateUTC.YearDay() {
			continue
		}

		stop, err := time.ParseInLocation("2006-01-02 15:04:05", item.EndTime, bangkok)
		if err != nil {
			continue
		}

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "gigatv.3bbtv.co.th",
			Title:   item.ProgramName,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		})
	}

	return programs, nil
}

type apiItem struct {
	ProgramName string `json:"programName"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
}

func (p *provider) Timezone() string { return "Asia/Bangkok" }
