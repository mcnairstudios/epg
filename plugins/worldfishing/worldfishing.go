package worldfishing

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

var eastern *time.Location

//go:embed channels.json
var channelsData []byte

func init() {
	var err error
	eastern, err = time.LoadLocation("America/New_York")
	if err != nil {
		eastern = time.FixedZone("EST", -5*60*60)
	}
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

func (p *provider) Name() string        { return "worldfishingnetwork.com" }
func (p *provider) Description() string { return "World Fishing Network TV schedule" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "worldfishingnetwork.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://www.worldfishingnetwork.com/schedule/77420?day=%s", date.Format("Mon"))

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

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	items := findShowItems(doc)
	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	var programs []domain.Program
	for _, item := range items {
		timeDiv := findByClass(item, "show-time")
		titleDiv := findByClass(item, "show-title")
		imgDiv := findByClass(item, "show-img")

		if timeDiv == nil || titleDiv == nil {
			continue
		}

		start := parseShowTime(timeDiv, date)
		if start.IsZero() {
			continue
		}

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				date = date.AddDate(0, 0, 1)
				start = parseShowTime(timeDiv, date)
			}
			prev.Stop = start
		}

		title := ""
		h3 := findFirst(titleDiv, "h3")
		if h3 != nil {
			title = strings.TrimSpace(extractText(h3))
		}

		subTitle := extractShowSubTitle(titleDiv)

		description := ""
		pNode := findFirst(titleDiv, "p")
		if pNode != nil {
			description = strings.TrimSpace(extractText(pNode))
		}

		image := ""
		if imgDiv != nil {
			img := findFirst(imgDiv, "img")
			if img != nil {
				src := getAttr(img, "src")
				if src != "" {
					if strings.HasPrefix(src, "//") {
						image = "https:" + src
					} else {
						image = src
					}
				}
			}
		}

		stop := start.Add(30 * time.Minute)
		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "worldfishingnetwork.com",
			Title:       title,
			SubTitle:    subTitle,
			Description: description,
			Image:       image,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	if programs == nil {
		return []domain.Program{}, nil
	}
	return programs, nil
}

func parseShowTime(timeDiv *html.Node, date time.Time) time.Time {
	h2 := findFirst(timeDiv, "h2")
	if h2 == nil {
		return time.Time{}
	}

	timeStr := strings.TrimSpace(extractDirectText(h2))

	span := findFirst(h2, "span")
	if span == nil {
		return time.Time{}
	}
	strong := findFirst(span, "strong")
	if strong == nil {
		return time.Time{}
	}
	period := strings.TrimSpace(extractText(strong))

	combined := fmt.Sprintf("%s %s %s", date.Format("2006-01-02"), timeStr, period)
	t, err := time.ParseInLocation("2006-01-02 3:04 PM", combined, eastern)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func extractShowSubTitle(titleDiv *html.Node) string {
	var sb strings.Builder
	for c := titleDiv.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		}
	}

	text := strings.TrimSpace(sb.String())
	if text != "" {
		return text
	}

	h3 := findFirst(titleDiv, "h3")
	if h3 == nil {
		return ""
	}
	for s := h3.NextSibling; s != nil; s = s.NextSibling {
		if s.Type == html.TextNode {
			t := strings.TrimSpace(s.Data)
			if t != "" {
				return t
			}
		}
	}
	return ""
}

func findShowItems(doc *html.Node) []*html.Node {
	var items []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			cls := getAttr(n, "class")
			if containsClass(cls, "show-item") {
				items = append(items, n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return items
}

func findByClass(n *html.Node, class string) *html.Node {
	if n.Type == html.ElementNode {
		cls := getAttr(n, "class")
		if containsClass(cls, class) {
			return n
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findByClass(c, class); found != nil {
			return found
		}
	}
	return nil
}

func containsClass(classList string, target string) bool {
	for _, c := range strings.Fields(classList) {
		if c == target {
			return true
		}
	}
	return false
}

func findFirst(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findFirst(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func extractDirectText(n *html.Node) string {
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		}
	}
	return sb.String()
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
