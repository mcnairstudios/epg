package tvmustra

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

var budapest *time.Location

//go:embed channels.json
var channelsData []byte

func init() {
	var err error
	budapest, err = time.LoadLocation("Europe/Budapest")
	if err != nil {
		panic(err)
	}
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "tvmustra.hu" }
func (p *provider) Description() string { return "Hungarian TV program guide tvmustra.hu" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvmustra.hu")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://www.tvmustra.hu/tvmusor/%s/%s", channel.SiteID, date.Format("2006-01-02"))

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
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return []domain.Program{}, nil
	}

	items := findProgramItems(doc)
	programs := make([]domain.Program, 0, len(items))

	for _, item := range items {
		timeStr := extractTime(item)
		title := extractTitle(item)

		if timeStr == "" || title == "" {
			continue
		}

		dateStr := date.Format("2006-01-02")
		start, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+timeStr, budapest)
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

		stop := start.Add(30 * time.Minute)

		programs = append(programs, domain.Program{
			Source: "tvmustra.hu",
			Title:  title,
			Start:  start,
			Stop:   stop,
		})
	}

	return programs, nil
}

func findProgramItems(n *html.Node) []*html.Node {
	var results []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" &&
			getAttr(n, "data-page") == "channel" && getAttr(n, "data-show") != "" {
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

func extractTime(n *html.Node) string {
	div := findElement(n, func(node *html.Node) bool {
		if node.Type != html.ElementNode || node.Data != "div" {
			return false
		}
		cls := getAttr(node, "class")
		return strings.HasPrefix(cls, "musor_lista_idopont")
	})
	if div != nil {
		return strings.TrimSpace(textContent(div))
	}
	return ""
}

func extractTitle(n *html.Node) string {
	div := findElement(n, func(node *html.Node) bool {
		if node.Type != html.ElementNode || node.Data != "div" {
			return false
		}
		cls := getAttr(node, "class")
		return strings.HasPrefix(cls, "musor_lista_cim")
	})
	if div != nil {
		return strings.TrimSpace(textContent(div))
	}
	return ""
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
