package digeagr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

var athensTZ *time.Location

//go:embed channels.json
var channelsData []byte

func init() {
	athensTZ, _ = time.LoadLocation("Europe/Athens")
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

func (p *provider) Name() string        { return "digea.gr" }
func (p *provider) Description() string { return "Digea Greek digital terrestrial TV" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "digea.gr")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	form := url.Values{}
	form.Set("action", "get_events")
	form.Set("date", fmt.Sprintf("%d-%d-%d", date.Year(), int(date.Month()), date.Day()))

	req, err := http.NewRequest(http.MethodPost, "https://www.digea.gr/el/api/epg/get-events", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

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
	var items []event
	if err := json.Unmarshal(data, &items); err != nil {
		return []domain.Program{}, nil
	}

	var programs []domain.Program
	for _, item := range items {
		if item.ChannelID != channel.SiteID {
			continue
		}

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "digea.gr",
			Title:       item.Title,
			Description: item.LongSynopsis,
			Lang:        channel.Lang,
		}

		if item.ActualTime != "" {
			if t, err := time.ParseInLocation("2006-01-02 15:04:05", item.ActualTime, athensTZ); err == nil {
				p.Start = t.UTC()
			}
		}
		if item.EndTime != "" {
			if t, err := time.ParseInLocation("2006-01-02 15:04:05", item.EndTime, athensTZ); err == nil {
				p.Stop = t.UTC()
			}
		}

		programs = append(programs, p)
	}

	if programs == nil {
		programs = []domain.Program{}
	}

	return programs, nil
}

type event struct {
	ChannelID    string `json:"channel_id"`
	Title        string `json:"title"`
	LongSynopsis string `json:"long_synopsis"`
	ActualTime   string `json:"actual_time"`
	EndTime      string `json:"end_time"`
}
