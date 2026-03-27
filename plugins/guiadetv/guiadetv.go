package guiadetv

import (
	"fmt"
	"io"
	"net/http"
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

func (p *provider) Name() string        { return "guiadetv.com" }
func (p *provider) Description() string { return "Guia de TV Brazilian TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "guiadetv.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://www.guiadetv.com/canal/%s", channel.SiteID)

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

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	items := findProgramRows(doc)
	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	dateStr := date.Format("2006-01-02")

	type rawProgram struct {
		title       string
		description string
		category    string
		start       time.Time
	}

	var raw []rawProgram

	for _, item := range items {
		dt := findDataDT(item)
		if dt == "" {
			continue
		}

		if !strings.HasPrefix(dt, dateStr) {
			continue
		}

		start := parseDT(dt)
		if start.IsZero() {
			continue
		}

		title := extractH3Text(item)
		if title == "" {
			continue
		}

		description, category := extractPInfo(item)

		raw = append(raw, rawProgram{
			title:       title,
			description: description,
			category:    category,
			start:       start,
		})
	}

	var programs []domain.Program
	for i, r := range raw {
		stop := r.start.Add(30 * time.Minute)
		if i < len(raw)-1 {
			stop = raw[i+1].start
		}
		if i > 0 {
			programs[i-1].Stop = r.start
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "guiadetv.com",
			Title:       r.title,
			Description: r.description,
			Category:    r.category,
			Lang:        channel.Lang,
			Start:       r.start,
			Stop:        stop,
		})
	}

	return programs, nil
}

func parseDT(dt string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05-07:00", dt)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func findDataDT(n *html.Node) string {
	if n.Type == html.ElementNode {
		if dt := getAttr(n, "data-dt"); dt != "" {
			return strings.TrimSpace(dt)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if dt := findDataDT(c); dt != "" {
			return dt
		}
	}
	return ""
}

func findProgramRows(n *html.Node) []*html.Node {
	var rows []*html.Node
	if n.Type == html.ElementNode && n.Data == "div" {
		cls := getAttr(n, "class")
		if strings.Contains(cls, "row") && strings.Contains(cls, "mt-1") {
			if findDataDT(n) != "" {
				rows = append(rows, n)
				return rows
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		rows = append(rows, findProgramRows(c)...)
	}
	return rows
}

func extractH3Text(n *html.Node) string {
	h3 := findFirstElement(n, "h3")
	if h3 == nil {
		return ""
	}
	return strings.TrimSpace(extractText(h3))
}

func extractPInfo(n *html.Node) (description, category string) {
	p := findFirstElement(n, "p")
	if p == nil {
		return "", ""
	}

	i := findFirstElement(p, "i")
	if i != nil {
		category = strings.TrimSpace(extractText(i))
	}

	var descParts []string
	for c := p.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			t := strings.TrimSpace(c.Data)
			if t != "" {
				descParts = append(descParts, t)
			}
		}
	}
	description = strings.Join(descParts, " ")

	return description, category
}

func findFirstElement(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findFirstElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func extractText(n *html.Node) string {
	if n == nil {
		return ""
	}
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
