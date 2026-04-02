package mitv

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
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{client: httpclient.Browser()})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "mi.tv" }
func (p *provider) Description() string { return "Mi.tv Latin American TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "mi.tv")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	parts := strings.SplitN(channel.SiteID, "#", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid site_id %q: expected country#channelId", channel.SiteID)
	}
	country, channelID := parts[0], parts[1]

	url := fmt.Sprintf("https://mi.tv/%s/async/channel/%s/%s/0",
		country, channelID, date.Format("2006-01-02"))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from mitv", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	content := string(data)
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}

	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil, err
	}

	listings := findNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "div" && hasAttr(n, "id", "listings")
	})
	if listings == nil {
		return nil, nil
	}

	ul := findNode(listings, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "ul"
	})
	if ul == nil {
		return nil, nil
	}

	var items []*html.Node
	for c := ul.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "li" {
			link := findNode(c, func(n *html.Node) bool {
				return n.Type == html.ElementNode && n.Data == "a" && hasClass(n, "program-link")
			})
			if link != nil {
				items = append(items, c)
			}
		}
	}

	currentDate := date.Truncate(24 * time.Hour).UTC()
	programs := make([]domain.Program, 0, len(items))

	for _, li := range items {
		timeStr := extractText(li, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "span" && hasClass(n, "time")
		})
		if timeStr == "" {
			continue
		}

		timeStr = strings.TrimSpace(timeStr)
		parts := strings.SplitN(timeStr, ":", 2)
		if len(parts) != 2 {
			continue
		}

		var hour, min int
		fmt.Sscanf(parts[0], "%d", &hour)
		fmt.Sscanf(parts[1], "%d", &min)

		start := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), hour, min, 0, 0, time.UTC)

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				currentDate = currentDate.Add(24 * time.Hour)
				start = time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), hour, min, 0, 0, time.UTC)
			}
			prev.Stop = start
		}

		stop := start.Add(1 * time.Hour)

		title := strings.TrimSpace(extractText(li, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "h2"
		}))

		category := strings.TrimSpace(extractText(li, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "span" && hasClass(n, "sub-title")
		}))

		description := strings.TrimSpace(extractText(li, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "p" && hasClass(n, "synopsis")
		}))

		image := extractImage(li)

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "mi.tv",
			Title:       title,
			Description: description,
			Category:    category,
			Image:       image,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

var bgImageRe = regexp.MustCompile(`background-image:\s*url\(['"]?(.*?)['"]?\)`)

func extractImage(n *html.Node) string {
	node := findNode(n, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "image")
	})
	if node == nil {
		return ""
	}
	style := getAttr(node, "style")
	if style == "" {
		return ""
	}
	m := bgImageRe.FindStringSubmatch(style)
	if len(m) < 2 {
		return ""
	}
	url := m[1]
	url = strings.Trim(url, `'"\\`)
	url = strings.ReplaceAll(url, `\'`, "'")
	url = strings.ReplaceAll(url, `\"`, `"`)
	url = strings.ReplaceAll(url, `\\`, `\`)
	return url
}

func findNode(n *html.Node, match func(*html.Node) bool) *html.Node {
	if match(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findNode(c, match); result != nil {
			return result
		}
	}
	return nil
}

func extractText(n *html.Node, match func(*html.Node) bool) string {
	node := findNode(n, match)
	if node == nil {
		return ""
	}
	return textContent(node)
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

func hasAttr(n *html.Node, key, val string) bool {
	for _, a := range n.Attr {
		if a.Key == key && a.Val == val {
			return true
		}
	}
	return false
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

func (p *provider) Timezone() string { return "America/Mexico_City" }
