package i24news

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

var jerusalemTZ = time.FixedZone("Asia/Jerusalem", 2*60*60)

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

func (p *provider) Name() string        { return "i24news.tv" }
func (p *provider) Description() string { return "i24NEWS international news channel" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "i24news.tv")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://api.i24news.tv/v2/%s/schedules", channel.SiteID)

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
		return nil, fmt.Errorf("HTTP %d from i24news", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	var items []scheduleItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}

	d := date.Truncate(24 * time.Hour).UTC()
	weekday := int(d.Weekday()) - 1
	if weekday < 0 {
		weekday = 6
	}

	programs := make([]domain.Program, 0)
	for _, item := range items {
		if item.Day != weekday {
			continue
		}
		if item.Show == nil {
			continue
		}

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "i24news.tv",
			Title:       item.Show.Title,
			Description: item.Show.Body,
			Lang:        channel.Lang,
		}

		if item.Show.Image != nil {
			p.Image = item.Show.Image.Href
		}

		if item.StartHour != "" {
			if t, err := time.ParseInLocation("2006-01-02 15:04", d.Format("2006-01-02")+" "+item.StartHour, jerusalemTZ); err == nil {
				p.Start = t.UTC()
			}
		}

		if item.EndHour != "" {
			if t, err := time.ParseInLocation("2006-01-02 15:04", d.Format("2006-01-02")+" "+item.EndHour, jerusalemTZ); err == nil {
				p.Stop = t.UTC()
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type scheduleItem struct {
	StartHour string `json:"startHour"`
	EndHour   string `json:"endHour"`
	Day       int    `json:"day"`
	Show      *show  `json:"show"`
}

type show struct {
	Title string     `json:"title"`
	Body  string     `json:"body"`
	Image *showImage `json:"image"`
}

type showImage struct {
	Href string `json:"href"`
}
