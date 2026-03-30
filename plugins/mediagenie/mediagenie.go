package mediagenie

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

var seoulTZ *time.Location

//go:embed channels.json
var channelsData []byte

var timeRe = regexp.MustCompile(`^\d{2}:\d{2}$`)

func init() {
	seoulTZ, _ = time.LoadLocation("Asia/Seoul")
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

func (p *provider) Name() string        { return "mediagenie.co.kr" }
func (p *provider) Description() string { return "MediaGenie Korean TV channels" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "mediagenie.co.kr")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://mediagenie.co.kr/%s/?qd=%s", channel.SiteID, date.Format("20060102"))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", "CUPID=d5ed6b77012aef2b4d4365ffd3a1a3a4")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from mediagenie", resp.StatusCode)
	}

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

	rows := findTableRows(doc)
	if len(rows) == 0 {
		return []domain.Program{}, nil
	}

	var programs []domain.Program
	for _, row := range rows {
		cells := extractRowCells(row)

		timeStr := cellText(cells, "col1")
		if !timeRe.MatchString(timeStr) {
			continue
		}

		start, err := time.ParseInLocation("2006-01-02 15:04", date.Format("2006-01-02")+" "+timeStr, seoulTZ)
		if err != nil {
			continue
		}

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				date = date.AddDate(0, 0, 1)
				start, _ = time.ParseInLocation("2006-01-02 15:04", date.Format("2006-01-02")+" "+timeStr, seoulTZ)
			}
			prev.Stop = start.UTC()
		}

		title := cellDirectText(cells, "col2")
		rating := cellText(cells, "col6")
		stop := start.Add(30 * time.Minute)

		prog := domain.Program{
			Channel:  channel.XmltvID,
			Source:   "mediagenie.co.kr",
			Title:    title,
			Lang:     channel.Lang,
			Start:    start.UTC(),
			Stop:     stop.UTC(),
		}

		if rating != "" {
			prog.Description = fmt.Sprintf("KMRB:%s", rating)
		}

		programs = append(programs, prog)
	}

	if programs == nil {
		return []domain.Program{}, nil
	}
	return programs, nil
}

type cellMap map[string]*html.Node

func findTableRows(doc *html.Node) []*html.Node {
	tbl := findByClass(doc, "tbl")
	if tbl == nil {
		return nil
	}

	tbody := findFirst(tbl, "tbody")
	if tbody == nil {
		return nil
	}

	var rows []*html.Node
	for c := tbody.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "tr" {
			rows = append(rows, c)
		}
	}
	return rows
}

func extractRowCells(row *html.Node) cellMap {
	m := make(cellMap)
	for c := row.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "td" {
			cls := getAttr(c, "class")
			for _, cl := range strings.Fields(cls) {
				m[cl] = c
			}
		}
	}
	return m
}

func cellText(m cellMap, class string) string {
	n, ok := m[class]
	if !ok {
		return ""
	}
	return strings.TrimSpace(extractText(n))
}

func cellDirectText(m cellMap, class string) string {
	n, ok := m[class]
	if !ok {
		return ""
	}
	return strings.TrimSpace(extractDirectText(n))
}

func findByClass(n *html.Node, class string) *html.Node {
	if n.Type == html.ElementNode {
		cls := getAttr(n, "class")
		for _, c := range strings.Fields(cls) {
			if c == class {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findByClass(c, class); found != nil {
			return found
		}
	}
	return nil
}

func findFirst(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findFirst(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func extractDirectText(n *html.Node) string {
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		}
	}
	return sb.String()
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
