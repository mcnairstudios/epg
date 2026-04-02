package nostv

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

const apiKey = "xe1dgrShwdR1DVOKGmsj8Ut4QLlGyOFI"

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

func (p *provider) Name() string        { return "nostv.pt" }
func (p *provider) Description() string { return "NOS TV Portugal" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "nostv.pt")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()
	url := fmt.Sprintf(
		"https://api.clg.nos.pt/nostv/ott/schedule/range/contents/guest?channels=%s&minDate=%sT00:00:00Z&maxDate=%sT23:59:59Z&isDateInclusive=true&client_id=%s",
		channel.SiteID, d.Format("2006-01-02"), d.Format("2006-01-02"), apiKey,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apikey", apiKey)
	req.Header.Set("Origin", "https://nostv.pt")
	req.Header.Set("Referer", "https://nostv.pt/")
	req.Header.Set("x-core-appversion", "2.20.2.2")
	req.Header.Set("x-core-contentratinglimit", "0")
	req.Header.Set("x-core-deviceid", "")
	req.Header.Set("x-core-devicetype", "web")
	req.Header.Set("x-core-timezoneoffset", "3600000")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from nostv", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var items []epgItem
	if err := json.Unmarshal(data, &items); err != nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "nostv.pt",
			Lang:    channel.Lang,
		}

		if item.Metadata != nil {
			p.Title = item.Metadata.Title
			p.Description = item.Metadata.Description
			p.Season = item.Metadata.Season
			p.Episode = item.Metadata.Episode
			if item.Metadata.SubTitle != "" {
				p.SubTitle = item.Metadata.SubTitle
			}
		}

		if len(item.Images) > 0 {
			p.Image = fmt.Sprintf(
				"https://mage.stream.nos.pt/mage/v1/Images?sourceUri=%s&profile=ott_1_452x340&client_id=%s",
				item.Images[0].URL, apiKey,
			)
		}

		if item.UtcDateTimeStart != "" {
			if t, err := time.Parse(time.RFC3339, item.UtcDateTimeStart); err == nil {
				p.Start = t.UTC()
			}
		}
		if item.UtcDateTimeEnd != "" {
			if t, err := time.Parse(time.RFC3339, item.UtcDateTimeEnd); err == nil {
				p.Stop = t.UTC()
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type epgItem struct {
	UtcDateTimeStart string    `json:"UtcDateTimeStart"`
	UtcDateTimeEnd   string    `json:"UtcDateTimeEnd"`
	Metadata         *metadata `json:"Metadata"`
	Images           []image   `json:"Images"`
}

type metadata struct {
	Title       string `json:"Title"`
	SubTitle    string `json:"SubTitle"`
	Description string `json:"Description"`
	Season      int    `json:"Season"`
	Episode     int    `json:"Episode"`
}

type image struct {
	URL string `json:"Url"`
}

func (p *provider) Timezone() string { return "Europe/Lisbon" }
