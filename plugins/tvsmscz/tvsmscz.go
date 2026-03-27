package tvsmscz

import (
	"bytes"
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
	"golang.org/x/net/html"
	"golang.org/x/text/encoding/charmap"
)

var prague *time.Location

//go:embed channels.json
var channelsData []byte

func init() {
	var err error
	prague, err = time.LoadLocation("Europe/Prague")
	if err != nil {
		panic(err)
	}
	plugin.Register(&provider{client: httpclient.Browser()})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "m.tv.sms.cz" }
func (p *provider) Description() string { return "Czech TV program guide via SMS.cz" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "m.tv.sms.cz")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Format("2006-01-02")
	url := fmt.Sprintf("https://m.tv.sms.cz/index.php?stanice=%s&cas=0&den=%s", channel.SiteID, d)

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

	return Parse(body, date)
}

func Parse(data []byte, date time.Time) ([]domain.Program, error) {
	decoded, err := charmap.Windows1250.NewDecoder().Bytes(data)
	if err != nil {
		decoded = data
	}

	doc, err := html.Parse(bytes.NewReader(decoded))
	if err != nil {
		return []domain.Program{}, nil
	}

	items := findProgramItems(doc)
	programs := make([]domain.Program, 0, len(items))

	for _, item := range items {
		timeStr := extractTimeSpan(item)
		title := extractProgramTitle(item)
		description := extractProgramDescription(item)

		if timeStr == "" || title == "" {
			continue
		}

		dateStr := date.Format("01/02/2006")
		start, err := time.ParseInLocation("01/02/2006 15.04", dateStr+" "+timeStr, prague)
		if err != nil {
			continue
		}
		start = start.UTC()

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				start = start.Add(24 * time.Hour)
				date = date.Add(24 * time.Hour)
			}
			prev.Stop = start
		}

		stop := start.Add(time.Hour)

		programs = append(programs, domain.Program{
			Source:      "m.tv.sms.cz",
			Title:       title,
			Description: description,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

func findProgramItems(n *html.Node) []*html.Node {
	container := findElement(n, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "div" && getAttr(node, "id") == "obsah"
	})
	if container == nil {
		return nil
	}

	var results []*html.Node
	var findItems func(*html.Node)
	findItems = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "porad") {
			results = append(results, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findItems(c)
		}
	}
	findItems(container)
	return results
}

func extractTimeSpan(n *html.Node) string {
	span := findElement(n, func(node *html.Node) bool {
		if node.Type != html.ElementNode || node.Data != "span" {
			return false
		}
		return node.Parent != nil && node.Parent.Data == "div"
	})
	if span != nil {
		return strings.TrimSpace(textContent(span))
	}
	return ""
}

func extractProgramTitle(n *html.Node) string {
	a := findElement(n, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "a" && hasClass(node, "nazev")
	})
	if a == nil {
		return ""
	}
	div := findFirstChildElement(a, "div")
	if div != nil {
		return strings.TrimSpace(textContent(div))
	}
	return ""
}

func extractProgramDescription(n *html.Node) string {
	a := findElement(n, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "a" && hasClass(node, "nazev")
	})
	if a == nil {
		return ""
	}
	detail := findElement(a, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "div" && hasClass(node, "detail")
	})
	if detail != nil {
		return strings.TrimSpace(textContent(detail))
	}
	return ""
}

func findFirstChildElement(n *html.Node, tag string) *html.Node {
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

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func findElement(n *html.Node, match func(*html.Node) bool) *html.Node {
	if match(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findElement(c, match); result != nil {
			return result
		}
	}
	return nil
}

func textContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var buf strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		buf.WriteString(textContent(c))
	}
	return buf.String()
}
