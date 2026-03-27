package i24news

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

func (p *provider) Name() string        { return "i24news.tv" }
func (p *provider) Description() string { return "i24NEWS international news channel" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "i24news-ar", Name: "i24NEWS Arabic", Source: "i24news.tv", SiteID: "ar", Lang: "ar", XmltvID: "I24NewsArabic.il"},
	}, nil
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

	loc, _ := time.LoadLocation("Asia/Jerusalem")
	weekday := int(date.Weekday()) - 1
	if weekday < 0 {
		weekday = 6
	}

	var programs []domain.Program
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
			p.Start = parseTime(date, item.StartHour, loc)
		}
		if item.EndHour != "" {
			p.Stop = parseTime(date, item.EndHour, loc)
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func parseTime(date time.Time, hourStr string, loc *time.Location) time.Time {
	s := date.Format("2006-01-02") + " " + hourStr
	t, err := time.ParseInLocation("2006-01-02 15:04", s, loc)
	if err != nil {
		return time.Time{}
	}
	return t
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
