package tvinsider2

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"

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

func (p *provider) Name() string        { return "tvinsider.com.v2" }
func (p *provider) Description() string { return "US TV Insider television listings (v2)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvinsider.com.v2")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	u := fmt.Sprintf("https://www.tvinsider.com/network/%s/schedule/", channel.SiteID)

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

	return Parse(body, channel, date)
}

var seasonEpisodeRe = regexp.MustCompile(`(?i)Season\s+(\d+)\s*•\s*Episode\s+(\d+)`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	items := parseItems(string(data), date)
	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}

	var programs []domain.Program
	for _, item := range items {
		start := parseTime(item.timeStr, date, loc)
		if start.IsZero() {
			continue
		}

		if len(programs) > 0 {
			programs[len(programs)-1].Stop = start
		}

		stop := start.Add(30 * time.Minute)

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tvinsider.com.v2",
			Title:       item.title,
			Description: item.description,
			Category:    item.category,
			Season:      item.season,
			Episode:     item.episode,
			SubTitle:    item.subtitle,
			Lang:        "en",
			Start:       start,
			Stop:        stop,
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type programItem struct {
	timeStr     string
	title       string
	description string
	category    string
	date        string
	season      int
	episode     int
	subtitle    string
}

func parseItems(content string, date time.Time) []programItem {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil
	}

	dateID := date.Format("01-02-2006")
	h2 := findNodeByID(doc, dateID)
	if h2 == nil {
		return nil
	}

	grid := nextElementSibling(h2)
	if grid == nil {
		return nil
	}

	var items []programItem
	for c := grid.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || c.Data != "a" {
			continue
		}
		item := parseAnchor(c)
		if item.title != "" {
			items = append(items, item)
		}
	}
	return items
}

func parseAnchor(a *html.Node) programItem {
	var item programItem

	for c := a.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		switch c.Data {
		case "div":
			if hasClass(c, "top-element") {
				timeNode := findChild(c, "time")
				if timeNode != nil {
					item.timeStr = strings.TrimSpace(getText(timeNode))
				}
			}
		case "h3":
			item.title = strings.TrimSpace(getText(c))
		case "h4":
			text := strings.TrimSpace(getText(c))
			parts := strings.SplitN(text, " \u2022 ", 2)
			if len(parts) >= 1 {
				item.category = strings.TrimSpace(parts[0])
			}
			if len(parts) >= 2 {
				item.date = strings.TrimSpace(parts[1])
			}
		case "h5":
			item.subtitle = strings.TrimSpace(getText(c))
		case "h6":
			text := strings.TrimSpace(getText(c))
			m := seasonEpisodeRe.FindStringSubmatch(text)
			if len(m) == 3 {
				item.season = atoi(m[1])
				item.episode = atoi(m[2])
			}
		case "p":
			item.description = strings.TrimSpace(getText(c))
		}
	}

	return item
}

func parseTime(timeStr string, date time.Time, loc *time.Location) time.Time {
	if timeStr == "" {
		return time.Time{}
	}
	dateStr := date.Format("2006-01-02") + " " + timeStr
	dateStr = strings.TrimSpace(dateStr)

	t, err := time.ParseInLocation("2006-01-02 3:04 PM", dateStr, loc)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func findNodeByID(n *html.Node, id string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == id {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findNodeByID(c, id); result != nil {
			return result
		}
	}
	return nil
}

func nextElementSibling(n *html.Node) *html.Node {
	for s := n.NextSibling; s != nil; s = s.NextSibling {
		if s.Type == html.ElementNode {
			return s
		}
	}
	return nil
}

func findChild(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			return c
		}
	}
	return nil
}

func hasClass(n *html.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range strings.Fields(a.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

func getText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(getText(c))
	}
	return sb.String()
}
