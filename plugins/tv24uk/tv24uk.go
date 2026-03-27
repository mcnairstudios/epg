package tv24uk

import (
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

func (p *provider) Name() string        { return "tv24.co.uk" }
func (p *provider) Description() string { return "TV24 UK TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tv24.co.uk")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Format("2006-01-02")
	url := fmt.Sprintf("https://tv24.co.uk/x/channel/%s/0/%s", channel.SiteID, d)

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

var timeRe = regexp.MustCompile(`^(\d{1,2}):(\d{2})(am|pm)$`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	items := findProgramNodes(doc)
	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	baseDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	var programs []domain.Program

	for _, item := range items {
		timeStr := extractText(findByClass(item, "time"))
		title := extractText(findNode(item, "h3"))
		desc := extractText(findNode(item, "p"))

		start := parseTimeStr(timeStr, baseDate)
		if start.IsZero() {
			continue
		}

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				baseDate = baseDate.Add(24 * time.Hour)
				start = parseTimeStr(timeStr, baseDate)
			}
			prev.Stop = start
		}

		stop := start.Add(30 * time.Minute)

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tv24.co.uk",
			Title:       title,
			Description: desc,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

func parseTimeStr(s string, baseDate time.Time) time.Time {
	m := timeRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return time.Time{}
	}

	hour := 0
	fmt.Sscanf(m[1], "%d", &hour)
	minute := 0
	fmt.Sscanf(m[2], "%d", &minute)

	if m[3] == "pm" && hour != 12 {
		hour += 12
	} else if m[3] == "am" && hour == 12 {
		hour = 0
	}

	return time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), hour, minute, 0, 0, time.UTC)
}

func findProgramNodes(n *html.Node) []*html.Node {
	var result []*html.Node
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && strings.Contains(a.Val, "program") {
				result = append(result, n)
				return result
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, findProgramNodes(c)...)
	}
	return result
}

func findByClass(n *html.Node, class string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && strings.Contains(a.Val, class) {
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
