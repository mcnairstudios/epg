package watchyourtv

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
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
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

func (p *provider) Name() string        { return "watchyour.tv" }
func (p *provider) Description() string { return "WatchYour.TV streaming guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "watchyour.tv")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	req, err := http.NewRequest(http.MethodGet, "https://www.watchyour.tv/guide.json", nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from watchyourtv", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	var guideData []channelData
	if err := json.Unmarshal(data, &guideData); err != nil {
		return []domain.Program{}, nil
	}

	var channelEntry *channelData
	for i := range guideData {
		if guideData[i].ID == channel.SiteID {
			channelEntry = &guideData[i]
			break
		}
	}

	if channelEntry == nil || len(channelEntry.Shows) == 0 {
		return []domain.Program{}, nil
	}

	dateStr := date.Format("2006-01-02")
	var programs []domain.Program
	for _, show := range channelEntry.Shows {
		if show.StartDay != dateStr {
			continue
		}

		tms, err := strconv.ParseInt(show.TMS, 10, 64)
		if err != nil {
			continue
		}
		start := time.Unix(tms, 0).UTC()

		dur, err := strconv.Atoi(show.Duration)
		if err != nil {
			continue
		}
		stop := start.Add(time.Duration(dur) * time.Minute)

		p := domain.Program{
			Channel:  channel.XmltvID,
			Source:   "watchyour.tv",
			Title:    show.Name,
			Image:    show.Icon,
			Category: show.Category,
			Lang:     channel.Lang,
			Start:    start,
			Stop:     stop,
		}
		programs = append(programs, p)
	}

	if programs == nil {
		return []domain.Program{}, nil
	}
	return programs, nil
}

type channelData struct {
	Name  string `json:"name"`
	ID    string `json:"id"`
	Shows []show `json:"shows"`
}

type show struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	StartDay string `json:"start_day"`
	Duration string `json:"duration"`
	Icon     string `json:"icon"`
	TMS      string `json:"tms"`
}

func (p *provider) Timezone() string { return "America/New_York" }
