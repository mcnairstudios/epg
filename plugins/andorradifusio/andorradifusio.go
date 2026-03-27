package andorradifusio

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
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

var madrid *time.Location

func init() {
	var err error
	madrid, err = time.LoadLocation("Europe/Madrid")
	if err != nil {
		madrid = time.FixedZone("CET", 1*60*60)
	}
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

func (p *provider) Name() string        { return "andorradifusio.ad" }
func (p *provider) Description() string { return "Andorra Difusio TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "andorradifusio.ad")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://www.andorradifusio.ad/programacio/%s", channel.SiteID)

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

type schedItem struct {
	Time  string
	Title string
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	dayStr := formatCatalanDay(date)
	items := extractSchedule(doc, dayStr)
	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	var programs []domain.Program
	for i, item := range items {
		start := parseTime(item.Time, date)
		if start.IsZero() {
			continue
		}

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				date = date.AddDate(0, 0, 1)
				start = parseTime(item.Time, date)
			}
			prev.Stop = start
		}

		stop := start.Add(1 * time.Hour)
		_ = i

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "andorradifusio.ad",
			Title:   item.Title,
			Lang:    channel.Lang,
			Start:   start,
			Stop:    stop,
		})
	}

	return programs, nil
}

func parseTime(timeStr string, date time.Time) time.Time {
	parts := strings.Split(strings.TrimSpace(timeStr), ":")
	if len(parts) != 2 {
		return time.Time{}
	}
	hour, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return time.Time{}
	}
	minute, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return time.Time{}
	}

	t := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, madrid)
	return t.UTC()
}

var catalanMonths = []string{
	"gener", "febrer", "març", "abril", "maig", "juny",
	"juliol", "agost", "setembre", "octubre", "novembre", "desembre",
}

func formatCatalanDay(date time.Time) string {
	day := date.Day()
	month := catalanMonths[date.Month()-1]
	return fmt.Sprintf("%02d %s", day, month[:3])
}

func extractSchedule(doc *html.Node, dayPrefix string) []schedItem {
	columns := findAllByClass(doc, "programacio-dia")
	for _, col := range columns {
		span := findByClass(col, "dia")
		if span == nil {
			continue
		}
		text := strings.TrimSpace(extractText(span))
		if text == dayPrefix+"." {
			return extractItems(col)
		}
	}
	return nil
}

func extractItems(col *html.Node) []schedItem {
	var h4s []*html.Node
	var ps []*html.Node
	collectH4P(col, &h4s, &ps)

	var items []schedItem
	for i := 0; i < len(h4s) && i < len(ps); i++ {
		items = append(items, schedItem{
			Time:  strings.TrimSpace(extractText(h4s[i])),
			Title: strings.TrimSpace(extractText(ps[i])),
		})
	}
	return items
}

func collectH4P(n *html.Node, h4s, ps *[]*html.Node) {
	if n.Type == html.ElementNode {
		if n.Data == "h4" {
			*h4s = append(*h4s, n)
			return
		}
		if n.Data == "p" {
			*ps = append(*ps, n)
			return
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectH4P(c, h4s, ps)
	}
}

func findAllByClass(n *html.Node, class string) []*html.Node {
	var result []*html.Node
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && classMatches(a.Val, class) {
				result = append(result, n)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, findAllByClass(c, class)...)
	}
	return result
}

func findByClass(n *html.Node, class string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && classMatches(a.Val, class) {
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

func classMatches(attrVal, class string) bool {
	for _, c := range strings.Fields(attrVal) {
		if c == class {
			return true
		}
	}
	return false
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
