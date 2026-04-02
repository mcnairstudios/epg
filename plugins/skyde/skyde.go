package skyde

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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

func (p *provider) Name() string        { return "sky.de" }
func (p *provider) Description() string { return "Sky Deutschland TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "sky.de")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := "https://www.sky.de/sgtvg/service/getBroadcastsForGrid"

	body := fmt.Sprintf(`{"cil":["%s"],"d":%d}`, channel.SiteID, date.UnixMilli())

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-GB")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from skyde", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	if !json.Valid(data) {
		return nil, fmt.Errorf("invalid JSON")
	}

	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return []domain.Program{}, nil
	}

	var channelData *channelEntry
	for i := range response.CL {
		if fmt.Sprintf("%v", response.CL[i].CI) == channel.SiteID {
			channelData = &response.CL[i]
			break
		}
	}

	if channelData == nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(channelData.EL))
	for _, item := range channelData.EL {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "sky.de",
			Title:       item.ET,
			Description: item.EPIT,
			Category:    item.EC,
			Lang:        channel.Lang,
			Start:       time.UnixMilli(item.BSDT).UTC(),
			Stop:        time.UnixMilli(item.BEDT).UTC(),
		}
		if item.SN != "" {
			if n, err := strconv.Atoi(item.SN); err == nil {
				p.Season = n
			}
		}
		if item.EN != "" {
			if n, err := strconv.Atoi(item.EN); err == nil {
				p.Episode = n
			}
		}
		if item.PU != "" {
			p.Image = "http://sky.de" + item.PU
		}
		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	CL []channelEntry `json:"cl"`
}

type channelEntry struct {
	CI json.Number `json:"ci"`
	EL []element   `json:"el"`
}

type element struct {
	BSDT int64  `json:"bsdt"`
	BEDT int64  `json:"bedt"`
	ET   string `json:"et"`
	EC   string `json:"ec"`
	EPIT string `json:"epit"`
	SN   string `json:"sn"`
	EN   string `json:"en"`
	PU   string `json:"pu"`
}

func (p *provider) Timezone() string { return "Europe/Berlin" }
