package turksatkablo

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

var istanbulTZ *time.Location

//go:embed channels.json
var channelsData []byte

func init() {
	istanbulTZ, _ = time.LoadLocation("Europe/Istanbul")
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

func (p *provider) Name() string        { return "turksatkablo.com.tr" }
func (p *provider) Description() string { return "Turksat Kablo TV Turkish cable provider" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "turksatkablo.com.tr")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	dayOfMonth := date.Format("02")
	url := fmt.Sprintf(
		"https://www.turksatkablo.com.tr/userUpload/EPG/%s.json?_=%d",
		dayOfMonth, date.UnixMilli(),
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
		return nil, fmt.Errorf("HTTP %d from turksatkablo.com.tr", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return []domain.Program{}, nil
	}

	var channelData *channelEntry
	for i := range response.Channels {
		if fmt.Sprintf("%v", response.Channels[i].ID) == channel.SiteID {
			channelData = &response.Channels[i]
			break
		}
	}

	if channelData == nil {
		return []domain.Program{}, nil
	}

	var programs []domain.Program
	dateStr := date.Format("2006-01-02")

	for _, item := range channelData.Programs {
		startStr := dateStr + " " + item.Start
		start, err := time.ParseInLocation("2006-01-02 15:04", startStr, istanbulTZ)
		if err != nil {
			continue
		}

		if len(programs) > 0 && start.Before(programs[len(programs)-1].Start) {
			start = start.Add(24 * time.Hour)
			dateStr = start.In(istanbulTZ).Format("2006-01-02")
		}

		stopStr := dateStr + " " + item.Stop
		stop, err := time.ParseInLocation("2006-01-02 15:04", stopStr, istanbulTZ)
		if err != nil {
			continue
		}

		if stop.Before(start) {
			stop = stop.Add(24 * time.Hour)
			dateStr = stop.In(istanbulTZ).Format("2006-01-02")
		}

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "turksatkablo.com.tr",
			Title:   item.Title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		})
	}

	if programs == nil {
		programs = []domain.Program{}
	}

	return programs, nil
}

type apiResponse struct {
	Channels []channelEntry `json:"k"`
}

type channelEntry struct {
	ID       json.Number `json:"x"`
	Index    int         `json:"i"`
	Name     string      `json:"n"`
	Programs []program   `json:"p"`
}

type program struct {
	ID    string `json:"a"`
	Title string `json:"b"`
	Start string `json:"c"`
	Stop  string `json:"d"`
}

func (p *provider) Timezone() string { return "Europe/Istanbul" }
