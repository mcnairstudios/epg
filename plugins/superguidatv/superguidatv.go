package superguidatv

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
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

var romeLocation *time.Location

func init() {
	romeLocation, _ = time.LoadLocation("Europe/Rome")
}

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "superguidatv.it" }
func (p *provider) Description() string { return "Super Guida TV Italy EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "superguidatv.it")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour).UTC()
	now := time.Now().UTC().Truncate(24 * time.Hour)
	diff := int(start.Sub(now).Hours() / 24)

	dayMap := map[int]string{
		0: "oggi",
		1: "domani",
		2: "dopodomani",
	}

	day, ok := dayMap[diff]
	if !ok {
		day = "oggi"
	}

	parts := strings.SplitN(channel.SiteID, "/", 2)
	slug := parts[0]
	rest := ""
	if len(parts) > 1 {
		rest = "/" + parts[1]
	}

	url := fmt.Sprintf(
		"https://www.superguidatv.it/programmazione-canale/%s/guida-programmi-tv-%s%s/",
		day, slug, rest,
	)

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

var categoryRe = regexp.MustCompile(`^([^(]+)`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return []domain.Program{}, nil
	}

	rows := findProgramRows(doc)
	programs := make([]domain.Program, 0, len(rows))
	currentDate := date

	for _, row := range rows {
		timeStr := extractTime(row)
		title := extractTitle(row)
		category := extractCategory(row)

		if timeStr == "" || title == "" {
			continue
		}

		dateStr := fmt.Sprintf("%s %s", currentDate.Format("2006-01-02"), timeStr)
		start, err := time.ParseInLocation("2006-01-02 15:04", dateStr, romeLocation)
		if err != nil {
			continue
		}

		if len(programs) > 0 && start.Before(programs[len(programs)-1].Start) {
			currentDate = currentDate.Add(24 * time.Hour)
			dateStr = fmt.Sprintf("%s %s", currentDate.Format("2006-01-02"), timeStr)
			start, err = time.ParseInLocation("2006-01-02 15:04", dateStr, romeLocation)
			if err != nil {
				continue
			}
		}

		if len(programs) > 0 {
			programs[len(programs)-1].Stop = start.UTC()
		}

		stop := start.Add(30 * time.Minute)

		programs = append(programs, domain.Program{
			Channel:  channel.XmltvID,
			Source:   "superguidatv.it",
			Title:    title,
			Category: category,
			Lang:     channel.Lang,
			Start:    start.UTC(),
			Stop:     stop.UTC(),
		})
	}

	return programs, nil
}

func findProgramRows(doc *html.Node) []*html.Node {
	container := findElement(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "sgtvchannelplan_divContainer")
	})
	if container == nil {
		return nil
	}

	var rows []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "sgtvchannelplan_divTableRow") {
			if hasDescendantWithID(n, "containerInfoEvent") {
				rows = append(rows, n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(container)
	return rows
}

func extractTime(row *html.Node) string {
	cell := findElement(row, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "sgtvchannelplan_hoursCell")
	})
	if cell == nil {
		return ""
	}
	return strings.TrimSpace(textContentExcluding(cell, "sgtvOnairSpan"))
}

func extractTitle(row *html.Node) string {
	span := findElement(row, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "span" && hasClass(n, "sgtvchannelplan_spanInfoNextSteps")
	})
	if span == nil {
		return ""
	}
	return strings.TrimSpace(textContent(span))
}

func extractCategory(row *html.Node) string {
	span := findElement(row, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "span" && hasClass(n, "sgtvchannelplan_spanEventType")
	})
	if span == nil {
		return ""
	}
	text := strings.TrimSpace(textContent(span))
	matches := categoryRe.FindStringSubmatch(text)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func hasDescendantWithID(n *html.Node, id string) bool {
	return findElement(n, func(node *html.Node) bool {
		if node.Type != html.ElementNode {
			return false
		}
		for _, a := range node.Attr {
			if a.Key == "id" && a.Val == id {
				return true
			}
		}
		return false
	}) != nil
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

func textContentExcluding(n *html.Node, excludeClass string) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	if n.Type == html.ElementNode && hasClass(n, excludeClass) {
		return ""
	}
	var buf strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		buf.WriteString(textContentExcluding(c, excludeClass))
	}
	return buf.String()
}
