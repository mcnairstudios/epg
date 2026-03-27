package indihome

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

func (p *provider) Name() string        { return "indihometv.com" }
func (p *provider) Description() string { return "IndiHome TV Indonesia guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "indihometv.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://www.indihometv.com/livetv/%s", channel.SiteID)

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

var jakarta *time.Location
var timeRangeRe = regexp.MustCompile(`(\d{2}:\d{2})\s*-\s*(\d{2}:\d{2})`)


func init() {
	jakarta, _ = time.LoadLocation("Asia/Jakarta")
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	dateStr := date.Format("2006-01-02")
	tabDiv := findByID(doc, "pills-"+dateStr)
	if tabDiv == nil {
		return []domain.Program{}, nil
	}

	items := findAllByClass(tabDiv, "schedule-item")
	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	baseDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, jakarta)
	var programs []domain.Program

	for _, item := range items {
		pNode := findNode(item, "p")
		bNode := findNode(item, "b")

		timeText := extractText(pNode)
		title := extractText(bNode)

		if title == "" {
			continue
		}

		m := timeRangeRe.FindStringSubmatch(timeText)
		if m == nil {
			continue
		}

		start := parseLocalTime(m[1], baseDate)
		stop := parseLocalTime(m[2], baseDate)

		if len(programs) > 0 && start.Before(programs[len(programs)-1].Start) {
			baseDate = baseDate.Add(24 * time.Hour)
			start = parseLocalTime(m[1], baseDate)
			stop = parseLocalTime(m[2], baseDate)
		}

		if stop.Before(start) {
			stop = stop.Add(24 * time.Hour)
		}

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "indihometv.com",
			Title:   title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		})
	}

	return programs, nil
}

func parseLocalTime(hhmm string, baseDate time.Time) time.Time {
	var hour, minute int
	fmt.Sscanf(hhmm, "%d:%d", &hour, &minute)
	return time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), hour, minute, 0, 0, jakarta)
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

func findAllByClass(n *html.Node, class string) []*html.Node {
	var result []*html.Node
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && strings.Contains(a.Val, class) {
				result = append(result, n)
				break
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, findAllByClass(c, class)...)
	}
	return result
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
