package kanil

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

var jerusalemTZ *time.Location

func init() {
	var err error
	jerusalemTZ, err = time.LoadLocation("Asia/Jerusalem")
	if err != nil {
		jerusalemTZ = time.FixedZone("Asia/Jerusalem", 2*60*60)
	}
}

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

func (p *provider) Name() string        { return "kan.org.il" }
func (p *provider) Description() string { return "KAN Israel public broadcasting" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "kan.org.il")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()

	url := fmt.Sprintf(
		"https://www.kan.org.il/tv-guide/tv_guidePrograms.ashx?stationID=%s&day=%s",
		channel.SiteID, d.Format("02/01/2006"),
	)

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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var items []kanItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "kan.org.il",
			Title:       item.Title,
			Description: item.LiveDesc,
			Lang:        channel.Lang,
		}

		if item.PictureCode != "" {
			p.Image = item.PictureCode
		}

		if item.StartTime != "" {
			if t, err := time.ParseInLocation("2006-01-02T15:04:05", item.StartTime, jerusalemTZ); err == nil {
				p.Start = t.UTC()
			}
		}

		if item.EndTime != "" {
			if t, err := time.ParseInLocation("2006-01-02T15:04:05", item.EndTime, jerusalemTZ); err == nil {
				p.Stop = t.UTC()
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type kanItem struct {
	Title       string `json:"title"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	LiveDesc    string `json:"live_desc"`
	PictureCode string `json:"picture_code"`
}
