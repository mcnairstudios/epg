package maxtvgo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

func (p *provider) Name() string        { return "maxtvgo.mk" }
func (p *provider) Description() string { return "MaxTV Go North Macedonia" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "maxtvgo.mk")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://prd-static-mkt.spectar.tv/rev-1636968171/client_api.php/epg/list/instance_id/1/language/mk/channel_id/%s/start/%s/stop/%s/include_current/true/format/json",
		channel.SiteID,
		date.Format("20060102150405"),
		date.AddDate(0, 0, 1).Format("20060102150405"),
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from maxtvgo", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(response.Programme))
	for _, item := range response.Programme {
		p := domain.Program{
			Channel:  channel.XmltvID,
			Source:   "maxtvgo.mk",
			Title:    item.Title,
			Category: item.Category,
			Lang:     channel.Lang,
		}

		if desc := parseDesc(item.Desc); desc != "" {
			p.Description = desc
		}
		if item.Icon.Attributes.Src != "" {
			p.Image = item.Icon.Attributes.Src
		}

		attrs := item.Attributes
		if attrs.Start != "" {
			if t, err := time.Parse("20060102150405 -0700", attrs.Start); err == nil {
				p.Start = t.UTC()
			}
		}
		if attrs.Stop != "" {
			if t, err := time.Parse("20060102150405 -0700", attrs.Stop); err == nil {
				p.Stop = t.UTC()
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func parseDesc(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "{}" || s == "null" {
		return ""
	}
	var desc string
	if err := json.Unmarshal(raw, &desc); err != nil {
		return ""
	}
	return desc
}

type apiResponse struct {
	Programme []programme `json:"programme"`
}

type programme struct {
	Attributes programmeAttrs `json:"@attributes"`
	Title      string          `json:"title"`
	Category   string          `json:"category"`
	Desc       json.RawMessage `json:"desc"`
	Icon       iconData        `json:"icon"`
}

type iconData struct {
	Attributes iconAttrs `json:"@attributes"`
}

type iconAttrs struct {
	Src string `json:"src"`
}

type programmeAttrs struct {
	Start string `json:"start"`
	Stop  string `json:"stop"`
}
