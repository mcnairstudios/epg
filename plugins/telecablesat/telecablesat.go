package telecablesat

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

var paris *time.Location

func init() {
	var err error
	paris, err = time.LoadLocation("Europe/Paris")
	if err != nil {
		panic(err)
	}

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

func (p *provider) Name() string        { return "tv-programme.telecablesat.fr" }
func (p *provider) Description() string { return "Telecablesat TV listings (France, 268 channels)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tv-programme.telecablesat.fr")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.In(paris).Format("2006-01-02")
	base := fmt.Sprintf(
		"https://tv-programme.telecablesat.fr/chaine/%s/index.html",
		channel.SiteID,
	)

	var allData [][]byte
	for _, period := range []string{"morning", "noon", "afternoon"} {
		url := fmt.Sprintf("%s?date=%s&period=%s", base, d, period)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0")

		resp, err := p.client.Do(req)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		allData = append(allData, body)
	}

	return Parse(allData, channel, date)
}

func Parse(pages [][]byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	var items []newsItem
	for _, page := range pages {
		items = append(items, extractItems(page)...)
	}

	if len(items) == 0 {
		return nil, nil
	}

	programs := make([]domain.Program, 0, len(items))
	currentDate := date

	for _, item := range items {
		start := parseTime(item.hour, currentDate)
		if start.IsZero() {
			continue
		}

		if len(programs) > 0 {
			prev := programs[len(programs)-1]
			if start.Before(prev.Start) {
				currentDate = currentDate.Add(24 * time.Hour)
				start = parseTime(item.hour, currentDate)
			}
			programs[len(programs)-1].Stop = start
		}

		stop := start.Add(time.Hour)

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tv-programme.telecablesat.fr",
			Title:       item.title,
			Description: item.description,
			Image:       item.image,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

type newsItem struct {
	hour        string
	title       string
	description string
	image       string
}

func parseTime(hourStr string, date time.Time) time.Time {
	hourStr = strings.TrimSpace(hourStr)
	if hourStr == "" {
		return time.Time{}
	}

	dateStr := date.In(paris).Format("2006-01-02")
	t, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+hourStr, paris)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func extractItems(data []byte) []newsItem {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil
	}

	nodes := findAllByClass(doc, "news")
	items := make([]newsItem, 0, len(nodes))

	for _, n := range nodes {
		item := newsItem{
			hour:        textByClass(n, "schedule-hour"),
			title:       textByClass(n, "title-left"),
			description: extractDescription(n),
			image:       extractImage(n),
		}
		if item.hour != "" {
			items = append(items, item)
		}
	}

	return items
}

func extractDescription(n *html.Node) string {
	itemContent := findByClass(n, "item-content")
	if itemContent == nil {
		return ""
	}
	p := findTag(itemContent, "p")
	if p == nil {
		return ""
	}
	return strings.TrimSpace(textContent(p))
}

func extractImage(n *html.Node) string {
	img := findTag(n, "img")
	if img == nil {
		return ""
	}
	src := attrVal(img, "src")
	if src == "" {
		return ""
	}
	if strings.HasPrefix(src, "//") {
		return "https:" + src
	}
	return src
}

func findAllByClass(n *html.Node, class string) []*html.Node {
	var results []*html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && hasClass(n, class) {
			results = append(results, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return results
}

func findByClass(n *html.Node, class string) *html.Node {
	var result *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if result != nil {
			return
		}
		if n.Type == html.ElementNode && hasClass(n, class) {
			result = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return result
}

func findTag(n *html.Node, tag string) *html.Node {
	var result *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if result != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == tag {
			result = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return result
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

func textByClass(n *html.Node, class string) string {
	el := findByClass(n, class)
	if el == nil {
		return ""
	}
	return strings.TrimSpace(textContent(el))
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

func attrVal(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
