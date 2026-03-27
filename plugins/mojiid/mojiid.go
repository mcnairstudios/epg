package mojiid

import (
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

var jakartaTZ = time.FixedZone("Asia/Jakarta", 7*60*60)

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

func (p *provider) Name() string        { return "moji.id" }
func (p *provider) Description() string { return "MOJI TV Indonesia" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "moji.id")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	req, err := http.NewRequest(http.MethodGet, "https://moji.id/schedule", nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	dateSlider := findByClass(doc, "date-slider")
	descSlider := findByClass(doc, "desc-slider")

	if dateSlider == nil || descSlider == nil {
		return []domain.Program{}, nil
	}

	monthDates := findAllByClass(dateSlider, "month")
	descSliders := findAllByClass(descSlider, "list-slider")

	targetMonth := date.Format("Jan 02")

	var dayIndex int = -1
	for i, md := range monthDates {
		text := strings.TrimSpace(extractText(md))
		if text == targetMonth {
			dayIndex = i
			break
		}
	}

	if dayIndex < 0 || dayIndex >= len(descSliders) {
		return []domain.Program{}, nil
	}

	slider := descSliders[dayIndex]
	accordions := findAllByClass(slider, "accordion")

	if len(accordions) == 0 {
		return []domain.Program{}, nil
	}

	year := date.Year()
	monthStr := date.Format("Jan")
	dayStr := fmt.Sprintf("%02d", date.Day())

	var programs []domain.Program
	for i, acc := range accordions {
		title := extractClassText(acc, "name-prog")
		desc := extractAccDescription(acc)
		timeStr := extractClassText(acc, "pkl")

		if timeStr == "" {
			continue
		}

		start := parseTime(year, monthStr, dayStr, timeStr)

		var stop time.Time
		if i+1 < len(accordions) {
			nextTime := extractClassText(accordions[i+1], "pkl")
			if nextTime != "" {
				stop = parseTime(year, monthStr, dayStr, nextTime)
			}
		}

		if stop.IsZero() {
			nextDay := fmt.Sprintf("%02d", date.Day()+1)
			stop = parseTime(year, monthStr, nextDay, "00:00")
		}

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "moji.id",
			Title:       title,
			Description: desc,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		}

		programs = append(programs, p)
	}

	if programs == nil {
		return []domain.Program{}, nil
	}

	return programs, nil
}

func parseTime(year int, month, day, timeStr string) time.Time {
	s := fmt.Sprintf("%d-%s-%s %s", year, month, day, timeStr)
	t, _ := time.ParseInLocation("2006-Jan-02 15:04", s, jakartaTZ)
	return t
}

func extractClassText(n *html.Node, class string) string {
	el := findByClass(n, class)
	if el == nil {
		return ""
	}
	return strings.TrimSpace(extractText(el))
}

func extractAccDescription(n *html.Node) string {
	contentAcc := findByClass(n, "content-acc")
	if contentAcc == nil {
		return ""
	}
	span := findNodeByTag(contentAcc, "span")
	if span == nil {
		return ""
	}
	return strings.TrimSpace(extractText(span))
}

func findAllByClass(n *html.Node, class string) []*html.Node {
	var result []*html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			cls := getAttr(node, "class")
			for _, c := range strings.Fields(cls) {
				if c == class {
					result = append(result, node)
					break
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return result
}

func findByClass(n *html.Node, class string) *html.Node {
	if n == nil {
		return nil
	}
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

func findNodeByTag(n *html.Node, tag string) *html.Node {
	if n == nil {
		return nil
	}
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findNodeByTag(c, tag); found != nil {
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

