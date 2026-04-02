package stod2

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

func (p *provider) Name() string        { return "stod2.is" }
func (p *provider) Description() string { return "Stod 2 Icelandic TV" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "stod2.is")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://api.stod2.is/dagskra/api/%s/%s", channel.SiteID, date.Format("2006-01-02"))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from stod2", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var items []scheduleItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		start, err := time.Parse(time.RFC3339, item.Upphaf)
		if err != nil {
			continue
		}
		stop := start.Add(time.Duration(item.Slott) * time.Minute)

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "stod2.is",
			Title:       item.IslTitill,
			SubTitle:    item.Undirtitill,
			Description: item.Lysing,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		}

		if item.Seria > 0 {
			p.Season = item.Seria
		}
		if item.Thattur > 0 {
			p.Episode = item.Thattur
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type scheduleItem struct {
	Upphaf      string `json:"upphaf"`
	IslTitill   string `json:"isltitill"`
	Undirtitill string `json:"undirtitill"`
	Lysing      string `json:"lysing"`
	Slott       int    `json:"slott"`
	Seria       int    `json:"seria"`
	Thattur     int    `json:"thattur"`
}

func (p *provider) Timezone() string { return "Atlantic/Reykjavik" }
