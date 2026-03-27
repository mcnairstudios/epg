package knrgl

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "knr.gl" }
func (p *provider) Description() string { return "KNR Greenland TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "knr.gl")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	u := "https://knr.gl/kl/tv/aallakaatitassat?ajax_form=1"

	formData := url.Values{}
	formData.Set("list_date", date.Format("2006-01-02"))
	formData.Set("form_id", "knr_radio_tv_program_overview_form")
	formData.Set("_triggering_element_name", "list_date")

	req, err := http.NewRequest(http.MethodPost, u, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	var entries []jsonEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	if len(entries) != 1 || entries[0].Data == "" {
		return []domain.Program{}, nil
	}

	items := parseItems(entries[0].Data)
	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	loc, err := time.LoadLocation("America/Nuuk")
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone: %w", err)
	}

	programs := make([]domain.Program, 0, len(items))
	for i, item := range items {
		dateStr := date.Format("2006-01-02")
		timeStr := fmt.Sprintf("%s %s", dateStr, item.time)
		start, err := time.ParseInLocation("2006-01-02 15:04", timeStr, loc)
		if err != nil {
			continue
		}

		stop := start.Add(1 * time.Hour)

		if i > 0 && len(programs) > 0 {
			programs[len(programs)-1].Stop = start.UTC()
		}

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "knr.gl",
			Title:       item.title,
			Description: item.description,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		}
		programs = append(programs, p)
	}

	return programs, nil
}

type jsonEntry struct {
	Data string `json:"data"`
}

type programItem struct {
	title       string
	description string
	time        string
}

func parseItems(htmlStr string) []programItem {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return nil
	}

	var items []programItem
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "li" && hasClass(n, "overview-program__list__item") {
			item := extractItem(n)
			if item.time != "" && item.title != "" {
				items = append(items, item)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return items
}

func extractItem(n *html.Node) programItem {
	var item programItem
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "span" {
			if hasClass(n, "overview-program__time") {
				item.time = strings.TrimSpace(textContent(n))
			} else if hasClass(n, "overview-program__text") {
				item.title = strings.TrimSpace(textContent(n))
			}
		}
		if n.Type == html.ElementNode && n.Data == "li" && hasClass(n, "overview-program__sublist__item") {
			text := strings.TrimSpace(textContent(n))
			text = strings.ReplaceAll(text, "\r\n", " ")
			text = strings.ReplaceAll(text, "\n", " ")
			text = strings.ReplaceAll(text, "\r", " ")
			item.description = text
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return item
}

func hasClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			for _, c := range strings.Fields(attr.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

func textContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textContent(c))
	}
	return sb.String()
}
