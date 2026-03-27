package tvboxbg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

var sofiaTZ, _ = time.LoadLocation("Europe/Sofia")

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

func (p *provider) Name() string        { return "tv.boxbg.net" }
func (p *provider) Description() string { return "Box TV Bulgaria" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tv.boxbg.net")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()
	url := fmt.Sprintf("https://tv.boxbg.net/channel/%s?day=%s", channel.SiteID, d.Format("02012006"))

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

var eventLinkRe = regexp.MustCompile(`eventLink\(\$event,(\{.*?\})\)`)

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	content := string(data)
	content = strings.ReplaceAll(content, "&quot;", "\"")
	content = strings.ReplaceAll(content, "&amp;", "&")
	content = strings.ReplaceAll(content, "&lt;", "<")
	content = strings.ReplaceAll(content, "&gt;", ">")

	matches := eventLinkRe.FindAllStringSubmatch(content, -1)
	programs := make([]domain.Program, 0)

	for _, m := range matches {
		if len(m) < 2 {
			continue
		}

		jsonStr := m[1]
		if !strings.HasSuffix(jsonStr, "}") {
			jsonStr += "}"
		}

		var item eventData
		if err := json.Unmarshal([]byte(jsonStr), &item); err != nil {
			continue
		}

		start, err := time.ParseInLocation("2006-01-02 15:04:05", item.StartAt, sofiaTZ)
		if err != nil {
			start, err = time.ParseInLocation("2006-01-02 15:04", item.StartAt, sofiaTZ)
			if err != nil {
				continue
			}
		}

		stop, err := time.ParseInLocation("2006-01-02 15:04:05", item.EndAt, sofiaTZ)
		if err != nil {
			stop, err = time.ParseInLocation("2006-01-02 15:04", item.EndAt, sofiaTZ)
			if err != nil {
				continue
			}
		}

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "tv.boxbg.net",
			Title:   item.Text,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		}

		if item.Categories != "" {
			cats := strings.Split(item.Categories, ",")
			if len(cats) > 0 {
				p.Category = cats[0]
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type eventData struct {
	Text       string `json:"text"`
	StartAt    string `json:"start_at"`
	EndAt      string `json:"end_at"`
	Categories string `json:"categories"`
}
