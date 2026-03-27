package magticom

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

var tbilisi *time.Location

func init() {
	var err error
	tbilisi, err = time.LoadLocation("Asia/Tbilisi")
	if err != nil {
		panic(err)
	}
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "magticom.ge" }
func (p *provider) Description() string { return "Magticom TV Guide (Georgia)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "magticom-260", Name: "Bollywood HD Russia", Source: "magticom.ge", SiteID: "260", Lang: "ka", XmltvID: "BollywoodHDRussia.ru"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour)
	end := start.Add(24 * time.Hour)

	form := url.Values{}
	form.Set("channelId", channel.SiteID)
	form.Set("start", fmt.Sprintf("%d", start.Unix()))
	form.Set("end", fmt.Sprintf("%d", end.Unix()))

	req, err := http.NewRequest(http.MethodPost, "https://www.magticom.ge/request/channel-program.php", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Referer", "https://www.magticom.ge/en/tv/tv-services/tv-guide")

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
	var items []apiItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		start, err := time.ParseInLocation("2006-01-02T15:04:05", item.StartTimestamp, tbilisi)
		if err != nil {
			continue
		}
		stop, err := time.ParseInLocation("2006-01-02T15:04:05", item.EndTimestamp, tbilisi)
		if err != nil {
			continue
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "magticom.ge",
			Title:       item.Title,
			Description: item.Info,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

type apiItem struct {
	Title          string `json:"title"`
	Info           string `json:"info"`
	StartTimestamp string `json:"startTimestamp"`
	EndTimestamp   string `json:"endTimestamp"`
}
