package stod2

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

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

func (p *provider) Name() string        { return "stod2.is" }
func (p *provider) Description() string { return "Stod 2 Icelandic TV" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "stod2", Name: "Stöð 2", Source: "stod2.is", SiteID: "STOD2", Lang: "is", XmltvID: "Stod2.is"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://api.stod2.is/dagskra/api/%s/%s", channel.SiteID, date.Format("2006-01-02"))

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
	var items []scheduleItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
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
