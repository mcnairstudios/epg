package osn

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
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

func (p *provider) Name() string        { return "osn.com" }
func (p *provider) Description() string { return "OSN TV schedule" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "osn-FTF", Name: "Fatafeat", Source: "osn.com", SiteID: "FTF", Lang: "ar", XmltvID: "Fatafeat.ae"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()
	encoded := url.QueryEscape(d.Format("01/02/2006"))

	u := fmt.Sprintf(
		"https://www.osn.com/api/TVScheduleWebService.asmx/time?dt=%s&co=AE&ch=%s&mo=false&hr=0",
		encoded, channel.SiteID,
	)

	req, err := http.NewRequest(http.MethodGet, u, nil)
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

var dubaiLocation = mustLoadLocation("Asia/Dubai")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var items []ScheduleItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		title := item.Title
		if channel.Lang == "ar" {
			title = item.ArabTitle
		}

		start, err := time.ParseInLocation("02 Jan 2006, 15:04", item.StartDateTime, dubaiLocation)
		if err != nil {
			continue
		}

		duration := int(math.Round(float64(item.TotalDivWidth) / 4.8))
		stop := start.Add(time.Duration(duration) * time.Minute)

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "osn.com",
			Title:   title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		})
	}

	return programs, nil
}

type ScheduleItem struct {
	Title         string `json:"Title"`
	ArabTitle     string `json:"Arab_Title"`
	TotalDivWidth int    `json:"TotalDivWidth"`
	StartDateTime string `json:"StartDateTime"`
}
