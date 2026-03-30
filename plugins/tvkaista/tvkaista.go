package tvkaista

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

var helsinki *time.Location

//go:embed channels.json
var channelsData []byte

var seasonEpisodeRe = regexp.MustCompile(`S(\d{2})E(\d{2})`)

func init() {
	var err error
	helsinki, err = time.LoadLocation("Europe/Helsinki")
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

func (p *provider) Name() string        { return "tvkaista.org" }
func (p *provider) Description() string { return "Finnish TV program guide tvkaista.org" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvkaista.org")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://www.tvkaista.org/%s/%s", channel.SiteID, date.Format("2006-01-02"))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from tvkaista", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return []domain.Program{}, nil
	}

	items := findListItems(doc)
	programs := make([]domain.Program, 0, len(items))

	for _, item := range items {
		timeRange := extractTimeRange(item)
		if timeRange == "" {
			continue
		}

		parts := strings.Split(timeRange, "-")
		if len(parts) != 2 {
			continue
		}

		startTime := strings.TrimSpace(parts[0])
		stopTime := strings.TrimSpace(parts[1])

		dateStr := date.Format("2006-01-02")
		start, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+startTime, helsinki)
		if err != nil {
			continue
		}
		stop, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+stopTime, helsinki)
		if err != nil {
			continue
		}

		if stop.Before(start) {
			stop = stop.Add(24 * time.Hour)
		}

		if len(programs) > 0 {
			prev := programs[len(programs)-1]
			if start.Before(prev.Start) {
				start = start.Add(24 * time.Hour)
				stop = stop.Add(24 * time.Hour)
				date = date.Add(24 * time.Hour)
			}
		} else {
			if start.Hour() > 18 {
				start = start.Add(-24 * time.Hour)
				stop = stop.Add(-24 * time.Hour)
				date = date.Add(-24 * time.Hour)
			}
		}

		title := extractTitle(item)
		desc := extractDescription(item)
		season, episode := extractSeasonEpisode(item)
		categories := extractCategories(item)

		p := domain.Program{
			Channel:     channel.XmltvID,
			Lang:        channel.Lang,
			Source:      "tvkaista.org",
			Title:       title,
			Description: desc,
			Season:      season,
			Episode:     episode,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		}

		if len(categories) > 0 {
			p.Category = categories[0]
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func findListItems(n *html.Node) []*html.Node {
	ul := findElement(n, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "ul" && hasClass(node, "list-group")
	})
	if ul == nil {
		return nil
	}

	var results []*html.Node
	for c := ul.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "li" {
			results = append(results, c)
		}
	}
	return results
}

func extractTimeRange(n *html.Node) string {
	div := findElement(n, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "div" &&
			hasClass(node, "bd-highlight") && hasClass(node, "me-2")
	})
	if div != nil {
		return strings.TrimSpace(textContent(div))
	}
	return ""
}

func extractTitle(n *html.Node) string {
	flexFill := findElement(n, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "div" &&
			hasClass(node, "bd-highlight") && hasClass(node, "flex-fill")
	})
	if flexFill == nil {
		return ""
	}

	span := findFirstChildByTag(flexFill, "span")
	if span != nil {
		return strings.TrimSpace(textContent(span))
	}
	return ""
}

func extractDescription(n *html.Node) string {
	collapse := findElement(n, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "div" && hasClass(node, "collapse")
	})
	if collapse == nil {
		return ""
	}

	p := findElement(collapse, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "p"
	})
	if p == nil {
		return ""
	}

	text := textContent(p)
	text = strings.ReplaceAll(text, "\n", "")
	text = collapseWhitespace(text)
	text = strings.ReplaceAll(text, "\u00a0", " ")
	return strings.TrimSpace(text)
}

func extractSeasonEpisode(n *html.Node) (int, int) {
	flexFill := findElement(n, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "div" &&
			hasClass(node, "bd-highlight") && hasClass(node, "flex-fill")
	})
	if flexFill == nil {
		return 0, 0
	}

	spans := findAllByTag(flexFill, "span")
	if len(spans) < 2 {
		return 0, 0
	}

	text := strings.TrimSpace(textContent(spans[1]))
	m := seasonEpisodeRe.FindStringSubmatch(text)
	if m == nil {
		return 0, 0
	}

	season, _ := strconv.Atoi(m[1])
	episode, _ := strconv.Atoi(m[2])
	return season, episode
}

func extractCategories(n *html.Node) []string {
	collapse := findElement(n, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "div" && hasClass(node, "collapse")
	})
	if collapse == nil {
		return nil
	}

	var categories []string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && hasClass(node, "badge") {
			text := strings.TrimSpace(textContent(node))
			if text != "" {
				categories = append(categories, text)
			}
			return
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(collapse)
	return categories
}

func collapseWhitespace(s string) string {
	var buf strings.Builder
	inSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' {
			if !inSpace {
				buf.WriteRune(' ')
				inSpace = true
			}
		} else {
			buf.WriteRune(r)
			inSpace = false
		}
	}
	return buf.String()
}

func findFirstChildByTag(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			return c
		}
	}
	return nil
}

func findAllByTag(n *html.Node, tag string) []*html.Node {
	var results []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			results = append(results, c)
		}
	}
	return results
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
