package musortv

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

func (p *provider) Name() string        { return "musor.tv" }
func (p *provider) Description() string { return "Hungarian TV program guide musor.tv" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "musor.tv")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	target := date.UTC().Truncate(24 * time.Hour)

	var url string
	if now.Equal(target) {
		url = fmt.Sprintf("https://musor.tv/mai/tvmusor/%s", channel.SiteID)
	} else {
		url = fmt.Sprintf("https://musor.tv/napi/tvmusor/%s/%s", channel.SiteID, date.Format("2006.01.02"))
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36 OPR/115.0.0.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body)
}

func Parse(data []byte) ([]domain.Program, error) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return []domain.Program{}, nil
	}

	items := findProgramEntries(doc)
	programs := make([]domain.Program, 0, len(items))

	for _, item := range items {
		timeAttr := extractTimeContent(item)
		title := extractTitle(item)

		if timeAttr == "" || title == "" {
			continue
		}

		normalized := strings.Replace(timeAttr, "GMT", "T", 1)
		normalized = strings.TrimSuffix(normalized, "Z")
		start, err := time.Parse("2006-01-02T15:04:05", normalized)
		if err != nil {
			continue
		}
		start = start.UTC()

		if len(programs) > 0 {
			programs[len(programs)-1].Stop = start
		}

		stop := start.Add(30 * time.Minute)

		p := domain.Program{
			Source:      "musor.tv",
			Title:       title,
			Description: extractDescription(item),
			Image:       extractImage(item),
			Start:       start,
			Stop:        stop,
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func findProgramEntries(n *html.Node) []*html.Node {
	var results []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "progentry_internal") {
			results = append(results, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return results
}

func extractTimeContent(n *html.Node) string {
	timeEl := findElement(n, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "time"
	})
	if timeEl != nil {
		return getAttr(timeEl, "content")
	}
	return ""
}

func extractTitle(n *html.Node) string {
	h3 := findElement(n, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "h3"
	})
	if h3 == nil {
		return ""
	}
	a := findElement(h3, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "a"
	})
	if a != nil {
		return strings.TrimSpace(textContent(a))
	}
	return ""
}

func extractDescription(n *html.Node) string {
	div := findElement(n, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "div" && hasClass(node, "progentrylong")
	})
	if div != nil {
		return strings.TrimSpace(textContent(div))
	}
	return ""
}

func extractImage(n *html.Node) string {
	div := findElement(n, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "div" && hasClass(node, "progentry_screenshot")
	})
	if div == nil {
		return ""
	}
	img := findElement(div, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "img"
	})
	if img != nil {
		src := getAttr(img, "src")
		if src != "" && !strings.HasPrefix(src, "http") {
			return "https:" + src
		}
		return src
	}
	return ""
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
