package royatv

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpcache"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{
		client: httpcache.Wrap(httpclient.Browser()),
	})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "roya-tv.com" }
func (p *provider) Description() string { return "Roya TV Jordan guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "roya-tv.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	diff := int(date.Sub(now).Hours() / 24)

	url := fmt.Sprintf(
		"https://backend.roya.tv/api/v01/channels/schedule-pagination?day_number=%d", diff)

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
		return nil, fmt.Errorf("HTTP %d from royatv", resp.StatusCode)
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

	dateStr := date.Format("2006-01-02")
	var dayData *dayEntry
	for i := range response.Data {
		if response.Data[i].Date == dateStr {
			dayData = &response.Data[i]
			break
		}
	}
	if dayData == nil {
		return []domain.Program{}, nil
	}

	var chData *channelEntry
	for i := range dayData.Channel {
		if fmt.Sprintf("%v", dayData.Channel[i].ID) == channel.SiteID {
			chData = &dayData.Channel[i]
			break
		}
	}
	if chData == nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(chData.Programs))
	for _, prog := range chData.Programs {
		start := time.Unix(prog.StartTimestamp, 0).UTC()
		stop := time.Unix(prog.EndTimestamp, 0).UTC()

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "roya-tv.com",
			Title:       prog.Name,
			Description: prog.Description,
			Image:       prog.ThumbnailWeb,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

type apiResponse struct {
	Data []dayEntry `json:"data"`
}

type dayEntry struct {
	Date    string         `json:"date"`
	Channel []channelEntry `json:"channel"`
}

type channelEntry struct {
	ID       json.Number    `json:"id"`
	Title    string         `json:"title"`
	Programs []programEntry `json:"programs"`
}

type programEntry struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	ThumbnailWeb   string `json:"thumbnail_web"`
	StartTimestamp int64  `json:"start_timestamp"`
	EndTimestamp   int64  `json:"end_timestamp"`
}

func (p *provider) Timezone() string { return "Asia/Amman" }
