package tivutv

import (
	"io"
	"net/http"
	"strconv"
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

var romeTZ, _ = time.LoadLocation("Europe/Rome")

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

func (p *provider) Name() string        { return "tivu.tv" }
func (p *provider) Description() string { return "Tivu Italian satellite TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tivu.tv")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	target := date.Truncate(24 * time.Hour)
	diff := int(target.Sub(now).Hours() / 24)

	u := "https://www.tivu.tv/epg_ajax_sat.aspx?d=" + strconv.Itoa(diff)

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

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	items := parseItems(string(data), channel.SiteID)
	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	var programs []domain.Program
	for _, item := range items {
		start := parseTime(item.timeStr, date)
		if start.IsZero() {
			continue
		}

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				start = start.Add(24 * time.Hour)
				date = date.Add(24 * time.Hour)
			}
			prev.Stop = start
		}

		stop := start.Add(30 * time.Minute)

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "tivu.tv",
			Title:   item.title,
			Lang:    channel.Lang,
			Start:   start,
			Stop:    stop,
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type programItem struct {
	title   string
	timeStr string
}

func parseItems(content string, siteID string) []programItem {
	if content == "" {
		return nil
	}

	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil
	}

	qNode := findQNode(doc, siteID)
	if qNode == nil {
		return nil
	}

	var items []programItem
	for c := qNode.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && hasClass(c, "p") {
			item := parsePNode(c)
			if item.title != "" {
				items = append(items, item)
			}
		}
	}
	return items
}

func findQNode(n *html.Node, siteID string) *html.Node {
	if n.Type == html.ElementNode && hasClass(n, "q") {
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == siteID {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findQNode(c, siteID); result != nil {
			return result
		}
	}
	return nil
}

func parsePNode(n *html.Node) programItem {
	var item programItem
	aNode := findChild(n, "a")
	if aNode == nil {
		return item
	}

	parts := renderAnchorParts(aNode)
	if len(parts) >= 1 {
		item.title = strings.TrimSpace(parts[0])
	}
	if len(parts) >= 3 {
		item.timeStr = strings.TrimSpace(parts[2])
	}

	return item
}

func renderAnchorParts(n *html.Node) []string {
	var parts []string
	var current strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "br" {
			parts = append(parts, current.String())
			current.Reset()
			continue
		}
		if c.Type == html.TextNode {
			current.WriteString(c.Data)
		}
	}
	parts = append(parts, current.String())
	return parts
}

func parseTime(timeStr string, date time.Time) time.Time {
	if timeStr == "" {
		return time.Time{}
	}
	dateStr := date.Format("2006-01-02") + " " + timeStr
	t, err := time.ParseInLocation("2006-01-02 15:04", dateStr, romeTZ)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
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
