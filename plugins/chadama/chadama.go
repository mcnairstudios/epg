package chadama

import (
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
)

//go:embed channels.json
var channelsData []byte

var casablanca *time.Location

func init() {
	casablanca, _ = time.LoadLocation("Africa/Casablanca")
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

func (p *provider) Name() string        { return "chada.ma" }
func (p *provider) Description() string { return "Chada TV Morocco guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "chada.ma")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := "https://chada.ma/fr/chada-tv/grille-tv/"

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
		return nil, fmt.Errorf("HTTP %d from chadama", resp.StatusCode)
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

	stopfix := findByID(doc, "stopfix")
	if stopfix == nil {
		return []domain.Program{}, nil
	}

	postsArea := findByClass(stopfix, "posts-area")
	if postsArea == nil {
		return []domain.Program{}, nil
	}

	h2s := findAllNodes(postsArea, "h2")
	var programs []domain.Program

	for _, h2 := range h2s {
		text := strings.TrimSpace(extractText(h2))
		if !strings.Contains(text, " - ") {
			continue
		}

		parts := strings.SplitN(text, " - ", 2)
		if len(parts) != 2 {
			continue
		}

		startStr := strings.TrimSpace(parts[0])
		stopStr := strings.TrimSpace(parts[1])

		start := parseTimeInDate(startStr, date)
		stop := parseTimeInDate(stopStr, date)
		if start.IsZero() || stop.IsZero() {
			continue
		}

		titleNode := findNextH3(h2)
		if titleNode == nil {
			continue
		}
		title := strings.TrimSpace(extractText(titleNode))
		if title == "" {
			continue
		}

		description := "No description available"
		descDiv := findNextSiblingDiv(titleNode)
		if descDiv != nil {
			desc := strings.TrimSpace(extractText(descDiv))
			if desc != "" {
				description = desc
			}
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "chada.ma",
			Title:       title,
			Description: description,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

func parseTimeInDate(timeStr string, date time.Time) time.Time {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return time.Time{}
	}
	hour, minute := 0, 0
	fmt.Sscanf(parts[0], "%d", &hour)
	fmt.Sscanf(parts[1], "%d", &minute)

	t := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, casablanca)
	return t.UTC()
}

func findNextH3(n *html.Node) *html.Node {
	for s := n.NextSibling; s != nil; s = s.NextSibling {
		if s.Type == html.ElementNode && s.Data == "h3" {
			return s
		}
	}
	return nil
}

func findNextSiblingDiv(n *html.Node) *html.Node {
	for s := n.NextSibling; s != nil; s = s.NextSibling {
		if s.Type == html.ElementNode && s.Data == "div" {
			return s
		}
	}
	return nil
}

func findByID(n *html.Node, id string) *html.Node {
	if n.Type == html.ElementNode && getAttr(n, "id") == id {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findByID(c, id); found != nil {
			return found
		}
	}
	return nil
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

func findAllNodes(n *html.Node, tag string) []*html.Node {
	var result []*html.Node
	if n.Type == html.ElementNode && n.Data == tag {
		result = append(result, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, findAllNodes(c, tag)...)
	}
	return result
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
