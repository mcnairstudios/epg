package tvheute

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

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

func (p *provider) Name() string        { return "tvheute.at" }
func (p *provider) Description() string { return "tvheute.at Austrian TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "tvheute-orf1", Name: "ORF1", Source: "tvheute.at", SiteID: "orf1", Lang: "de", XmltvID: "ORF1.at"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Format("02-01-2006")
	url := fmt.Sprintf("https://tvheute.at/part/channel-shows/partial/%s/%s", channel.SiteID, d)

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

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	container := findByID(doc, "showListContainer")
	if container == nil {
		return []domain.Program{}, nil
	}

	rows := findAllNodes(container, "tr")
	programs := make([]domain.Program, 0, len(rows))

	for _, row := range rows {
		durationWrapper := findByClass(row, "duration-wrapper")
		if durationWrapper == nil {
			continue
		}

		startStr := getAttr(durationWrapper, "data-start")
		stopStr := getAttr(durationWrapper, "data-stop")

		start := parseISO(startStr)
		stop := parseISO(stopStr)

		if start.IsZero() || stop.IsZero() {
			continue
		}

		titleCol := findByClass(row, "title-col")
		title := ""
		if titleCol != nil {
			strong := findNode(titleCol, "strong")
			if strong != nil {
				title = extractText(strong)
			}
		}

		if title == "" {
			continue
		}

		desc := ""
		if titleCol != nil {
			descDiv := findByClass(titleCol, "description")
			if descDiv != nil {
				desc = extractText(descDiv)
			}
		}

		category := ""
		stationCol := findByClass(row, "station-col")
		if stationCol != nil {
			typeSpan := findByClass(stationCol, "type")
			if typeSpan != nil {
				category = extractText(typeSpan)
			}
		}

		image := ""
		if titleCol != nil {
			img := findNode(titleCol, "img")
			if img != nil {
				src := getAttr(img, "data-src-desktop")
				if src != "" {
					image = "https://tvheute.at" + src
				}
			}
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tvheute.at",
			Title:       title,
			Description: desc,
			Category:    category,
			Image:       image,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		})
	}

	return programs, nil
}

func parseISO(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05-07:00", s)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}

func findByID(n *html.Node, id string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == id {
				return n
			}
		}
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
		for _, a := range n.Attr {
			if a.Key == "class" && containsClass(a.Val, class) {
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

func containsClass(classList, target string) bool {
	for _, c := range strings.Fields(classList) {
		if c == target {
			return true
		}
	}
	return false
}

func findNode(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findNode(c, tag); found != nil {
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

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
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
	return strings.TrimSpace(sb.String())
}
