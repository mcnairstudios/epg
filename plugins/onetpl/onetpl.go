package onetpl

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

func (p *provider) Name() string        { return "programtv.onet.pl" }
func (p *provider) Description() string { return "Onet.pl Polish TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "onetpl-13thstreet", Name: "13th Street", Source: "programtv.onet.pl", SiteID: "13th-street-250", Lang: "pl", XmltvID: "13thStreet.de"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	target := date.Truncate(24 * time.Hour).UTC()
	day := int(target.Sub(now).Hours() / 24)

	url := fmt.Sprintf("https://programtv.onet.pl/program-tv/%s?dzien=%d", channel.SiteID, day)

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

var warsaw *time.Location

func init() {
	warsaw, _ = time.LoadLocation("Europe/Warsaw")
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	emissionsUL := findEmissionsUL(doc)
	if emissionsUL == nil {
		return []domain.Program{}, nil
	}

	items := findAllDirectChildren(emissionsUL, "li")
	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	baseDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	var programs []domain.Program

	for _, item := range items {
		hourSpan := findByClass(item, "hour")
		timeStr := extractText(hourSpan)

		titlesDiv := findByClass(item, "titles")
		title := ""
		category := ""
		desc := ""

		if titlesDiv != nil {
			a := findNode(titlesDiv, "a")
			if a != nil {
				title = extractText(a)
			}

			typeSpan := findByClass(titlesDiv, "type")
			if typeSpan != nil {
				category = extractText(typeSpan)
			}

			p := findNode(titlesDiv, "p")
			if p != nil {
				desc = strings.TrimSpace(extractText(p))
			}
		}

		if title == "" {
			continue
		}

		start := parseTimeInWarsaw(timeStr, baseDate)
		if start.IsZero() {
			continue
		}

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				baseDate = baseDate.Add(24 * time.Hour)
				start = parseTimeInWarsaw(timeStr, baseDate)
			}
			prev.Stop = start
		}

		stop := start.Add(1 * time.Hour)

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "programtv.onet.pl",
			Title:       title,
			Description: desc,
			Category:    category,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

func parseTimeInWarsaw(timeStr string, baseDate time.Time) time.Time {
	timeStr = strings.TrimSpace(timeStr)
	if timeStr == "" {
		return time.Time{}
	}

	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return time.Time{}
	}

	var hour, minute int
	fmt.Sscanf(parts[0], "%d", &hour)
	fmt.Sscanf(parts[1], "%d", &minute)

	local := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), hour, minute, 0, 0, warsaw)
	return local.UTC()
}

func findEmissionsUL(n *html.Node) *html.Node {
	div := findByClass(n, "emissions")
	if div == nil {
		return nil
	}
	return findNode(div, "ul")
}

func findAllDirectChildren(n *html.Node, tag string) []*html.Node {
	var result []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			result = append(result, c)
		}
	}
	return result
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
