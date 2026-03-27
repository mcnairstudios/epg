package myafn

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

func (p *provider) Name() string        { return "v3.myafn.dodmedia.osd.mil" }
func (p *provider) Description() string { return "American Forces Network schedule" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "v3.myafn.dodmedia.osd.mil")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.UTC().Format("2006-01-02")
	url := fmt.Sprintf(
		"https://v3.myafn.dodmedia.osd.mil/api/json/32/%s@0000/%s@2359/schedule.json",
		d, d,
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
	var items []apiItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, nil
	}

	programs := make([]domain.Program, 0)
	for _, item := range items {
		if fmt.Sprint(item.B) != channel.SiteID {
			continue
		}

		start := parseTime(item.E)
		if start.IsZero() {
			continue
		}
		stop := start.Add(time.Duration(item.O) * time.Minute)

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "v3.myafn.dodmedia.osd.mil",
			Title:       item.H,
			SubTitle:    item.I,
			Description: item.L,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		}

		if item.M != "" {
			p.Category = item.M
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func parseTime(s string) time.Time {
	parts := strings.Split(s, ",")
	if len(parts) < 6 {
		return time.Time{}
	}

	vals := make([]int, len(parts))
	for i, p := range parts {
		v, err := strconv.Atoi(p)
		if err != nil {
			return time.Time{}
		}
		vals[i] = v
	}

	return time.Date(vals[0], time.Month(vals[1]+1), vals[2], vals[3], vals[4], vals[5], 0, time.UTC)
}

type apiItem struct {
	B int    `json:"b"`
	E string `json:"e"`
	H string `json:"h"`
	I string `json:"i"`
	J string `json:"j"`
	L string `json:"l"`
	M string `json:"m"`
	O int    `json:"o"`
}
