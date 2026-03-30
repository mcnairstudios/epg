package cytacy

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

func (p *provider) Name() string        { return "cyta.com.cy" }
func (p *provider) Description() string { return "Cyta Cyprus TV provider" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "cyta.com.cy")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour).UTC()
	end := start.Add(24 * time.Hour)

	url := fmt.Sprintf(
		"https://epg.cyta.com.cy/api/mediacatalog/fetchEpg?startTimeEpoch=%d&endTimeEpoch=%d&language=1&channelIds=%s",
		start.UnixMilli(), end.UnixMilli(), channel.SiteID,
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
		return nil, fmt.Errorf("HTTP %d from cytacy", resp.StatusCode)
	}

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

	var programs []domain.Program
	for _, ch := range response.ChannelEpgs {
		for _, epg := range ch.EpgPlayables {
			startTime := time.UnixMilli(epg.StartTime).UTC()
			stopTime := time.UnixMilli(epg.EndTime).UTC()

			programs = append(programs, domain.Program{
				Channel: channel.XmltvID,
				Source:  "cyta.com.cy",
				Title:   epg.Name,
				Lang:    channel.Lang,
				Start:   startTime,
				Stop:    stopTime,
			})
		}
	}

	if programs == nil {
		programs = []domain.Program{}
	}

	return programs, nil
}

type apiResponse struct {
	ChannelEpgs []channelEpg `json:"channelEpgs"`
}

type channelEpg struct {
	EpgPlayables []epgPlayable `json:"epgPlayables"`
}

type epgPlayable struct {
	Name      string `json:"name"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
}
