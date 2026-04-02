package vodafonept

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
	plugin.Register(&provider{client: httpclient.Browser()})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "vodafone.pt" }
func (p *provider) Description() string { return "Vodafone Portugal EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "vodafone.pt")
}

var periods = []string{"00-06", "06-12", "12-18", "18-00"}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	lisbon, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		return nil, err
	}

	d := date.In(lisbon)
	dateStr := fmt.Sprintf("%d/%02d/%02d", d.Year(), d.Month(), d.Day())

	var allItems []epgObject
	for _, period := range periods {
		url := fmt.Sprintf("https://cdn.pt.vtv.vodafone.com/epg/%s/%s/%s", channel.SiteID, dateStr, period)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Origin", "https://www.vodafone.pt")
		req.Header.Set("Referer", "https://www.vodafone.pt/")
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; tv_grab_pt_vodafone)")
		req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
		req.Header.Set("Accept-Language", "pt-PT,pt;q=0.9,en;q=0.8")

		resp, err := p.client.Do(req)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		items := parseItems(body)
		allItems = append(allItems, items...)
	}

	return buildPrograms(allItems, channel), nil
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	items := parseItems(data)
	return buildPrograms(items, channel), nil
}

func ParseMulti(allData [][]byte, channel domain.Channel) ([]domain.Program, error) {
	var allItems []epgObject
	for _, data := range allData {
		allItems = append(allItems, parseItems(data)...)
	}
	return buildPrograms(allItems, channel), nil
}

func buildPrograms(items []epgObject, channel domain.Channel) []domain.Program {
	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		if item.StartDate == 0 || item.EndDate == 0 {
			continue
		}

		start := time.Unix(item.StartDate, 0).UTC()
		stop := time.Unix(item.EndDate, 0).UTC()
		if stop.Before(start) {
			stop = stop.Add(24 * time.Hour)
		}

		name := item.Name
		if name == "" {
			name = "Sem título"
		}

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "vodafone.pt",
			Title:   name,
			Lang:    channel.Lang,
			Start:   start,
			Stop:    stop,
		}

		if item.Description != "" {
			p.Description = item.Description
		}

		programs = append(programs, p)
	}
	return programs
}

func parseItems(data []byte) []epgObject {
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil
	}
	if resp.Result == nil {
		return nil
	}
	return resp.Result.Objects
}

type apiResponse struct {
	Result *resultData `json:"result"`
}

type resultData struct {
	Objects []epgObject `json:"objects"`
}

type epgObject struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	StartDate   int64  `json:"startDate"`
	EndDate     int64  `json:"endDate"`
}

func (p *provider) Timezone() string { return "Europe/Lisbon" }
