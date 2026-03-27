package cgateslt

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

var vilniusTZ, _ = time.LoadLocation("Europe/Vilnius")

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

func (p *provider) Name() string        { return "cgates.lt" }
func (p *provider) Description() string { return "Cgates Lithuania TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "cgates-lrt", Name: "LRT televizija", Source: "cgates.lt", SiteID: "lrt-televizija-hd", Lang: "lt", XmltvID: "LRTTV.lt"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://www.cgates.lt/tv-kanalai/%s/", channel.SiteID)

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

	return Parse(body, channel, date)
}

var timeRe = regexp.MustCompile(`(\d{2}:\d{2})`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return []domain.Program{}, nil
	}

	d := date.Truncate(24 * time.Hour).UTC()
	dateStr := d.Format("2006-01-02")

	section := findDateSection(doc, dateStr)
	if section == nil {
		return []domain.Program{}, nil
	}

	rows := findProgramRows(section)
	programs := make([]domain.Program, 0)

	for _, row := range rows {
		timeText := extractTimeFromRow(row)
		title := extractTitleFromRow(row)
		desc := extractDescFromRow(row)

		if timeText == "" || title == "" {
			continue
		}

		start, err := time.ParseInLocation("2006-01-02 15:04", d.Format("2006-01-02")+" "+timeText, vilniusTZ)
		if err != nil {
			continue
		}

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				d = d.Add(24 * time.Hour)
				start, _ = time.ParseInLocation("2006-01-02 15:04", d.Format("2006-01-02")+" "+timeText, vilniusTZ)
			}
			prev.Stop = start.UTC()
		}

		stop := start.Add(30 * time.Minute)

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "cgates.lt",
			Title:       title,
			Description: desc,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func findDateSection(n *html.Node, dateStr string) *html.Node {
	if n.Type == html.ElementNode && hasClass(n, "dt-fancy-title") {
		text := collectText(n)
		if strings.Contains(text, dateStr) {
			separator := n.Parent
			if separator != nil {
				container := separator.Parent
				if container != nil {
					return container
				}
			}
			return n
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findDateSection(c, dateStr); result != nil {
			return result
		}
	}
	return nil
}

func findProgramRows(n *html.Node) []*html.Node {
	var rows []*html.Node
	if n.Type == html.ElementNode && n.Data == "table" && hasClass(n, "tv_programa") {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "tbody" {
				for tr := c.FirstChild; tr != nil; tr = tr.NextSibling {
					if tr.Type == html.ElementNode && tr.Data == "tr" {
						rows = append(rows, tr)
					}
				}
			}
			if c.Type == html.ElementNode && c.Data == "tr" {
				rows = append(rows, c)
			}
		}
		return rows
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		rows = append(rows, findProgramRows(c)...)
	}
	return rows
}

func extractTimeFromRow(n *html.Node) string {
	nodes := findNodesByClass(n, "laikas")
	for _, node := range nodes {
		text := strings.TrimSpace(collectText(node))
		m := timeRe.FindString(text)
		if m != "" {
			return m
		}
	}
	return ""
}

func extractTitleFromRow(n *html.Node) string {
	tds := findDirectElements(n, "td")
	if len(tds) >= 2 {
		toggleTitles := findNodesByClass(tds[1], "vc_toggle_title")
		if len(toggleTitles) > 0 {
			text := strings.TrimSpace(collectText(toggleTitles[0]))
			if text != "" {
				return text
			}
		}
		return strings.TrimSpace(collectText(tds[1]))
	}
	return ""
}

func extractDescFromRow(n *html.Node) string {
	nodes := findNodesByClass(n, "vc_toggle_content")
	for _, node := range nodes {
		ps := findElements(node, "p")
		for _, p := range ps {
			text := strings.TrimSpace(collectText(p))
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func hasClass(n *html.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" && strings.Contains(a.Val, class) {
			return true
		}
	}
	return false
}

func findNodesByClass(n *html.Node, class string) []*html.Node {
	var results []*html.Node
	if n.Type == html.ElementNode && hasClass(n, class) {
		results = append(results, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		results = append(results, findNodesByClass(c, class)...)
	}
	return results
}

func findDirectElements(n *html.Node, tag string) []*html.Node {
	var results []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			results = append(results, c)
		}
	}
	return results
}

func findElements(n *html.Node, tag string) []*html.Node {
	var results []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			results = append(results, c)
		}
		results = append(results, findElements(c, tag)...)
	}
	return results
}

func collectText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(collectText(c))
	}
	return sb.String()
}
