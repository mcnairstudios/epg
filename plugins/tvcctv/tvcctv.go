package tvcctv

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

func (p *provider) Name() string        { return "tv.cctv.com" }
func (p *provider) Description() string { return "CCTV Chinese Central Television" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tv.cctv.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://api.cntv.cn/epg/getEpgInfoByChannelNew?serviceId=tvcctv&c=%s&d=%s",
		channel.SiteID,
		date.Format("20060102"),
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

	channelData, ok := response.Data[channel.SiteID]
	if !ok {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(channelData.List))
	for _, item := range channelData.List {
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "tv.cctv.com",
			Title:   item.Title,
			Lang:    channel.Lang,
			Start:   time.Unix(item.StartTime, 0).UTC(),
			Stop:    time.Unix(item.EndTime, 0).UTC(),
		}
		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Data map[string]channelData `json:"data"`
}

type channelData struct {
	List []programItem `json:"list"`
}

type programItem struct {
	Title     string `json:"title"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
}
