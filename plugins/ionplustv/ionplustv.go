package ionplustv

import (
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
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

//go:embed channels.json
var channelsData []byte

var (
	eastern    *time.Location
	durationRe = regexp.MustCompile(`\((\d+)`)
	ratingRe   = regexp.MustCompile(`^([^(]+)`)
)

func init() {
	eastern, _ = time.LoadLocation("America/New_York")
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

func (p *provider) Name() string        { return "ionplustv.com" }
func (p *provider) Description() string { return "ION Plus TV schedule" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "ionplustv.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://ionplustv.com/schedule/%s", date.Format("2006-01-02"))

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

	accordion := findByID(doc, "accordion")
	if accordion == nil {
		return []domain.Program{}, nil
	}

	var panels []*html.Node
	for c := accordion.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "div" {
			panels = append(panels, c)
		}
	}

	if len(panels) == 0 {
		return []domain.Program{}, nil
	}

	curDate := date
	var programs []domain.Program

	for _, panel := range panels {
		title := extractByClass(panel, "show-title")
		if title == "" {
			continue
		}

		timeStr, meridiem := extractTime(panel)
		if timeStr == "" {
			continue
		}
		if !strings.Contains(timeStr, ":") {
			timeStr += ":00"
		}

		start := parseAMPMTime(timeStr, meridiem, curDate)

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				curDate = curDate.Add(24 * time.Hour)
				start = parseAMPMTime(timeStr, meridiem, curDate)
			}
			prev.Stop = start
		}

		subTitle := extractSubTitle(panel)
		description := extractDescription(panel)
		image := extractImage(panel)
		ratingText := extractByClass(panel, "tv-rating")

		var duration int
		if matches := durationRe.FindStringSubmatch(ratingText); matches != nil {
			duration, _ = strconv.Atoi(matches[1])
		}

		stop := start.Add(time.Duration(duration) * time.Minute)

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "ionplustv.com",
			Title:       title,
			SubTitle:    subTitle,
			Description: description,
			Image:       image,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

func parseAMPMTime(timeStr, meridiem string, date time.Time) time.Time {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return time.Time{}
	}
	h, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	m, _ := strconv.Atoi(strings.TrimSpace(parts[1]))

	meridiem = strings.ToUpper(strings.TrimSpace(meridiem))
	if meridiem == "PM" && h != 12 {
		h += 12
	} else if meridiem == "AM" && h == 12 {
		h = 0
	}

	return time.Date(date.Year(), date.Month(), date.Day(), h, m, 0, 0, eastern).UTC()
}

func extractTime(panel *html.Node) (string, string) {
	panelTitle := findByClass(panel, "panel-title")
	if panelTitle == nil {
		return "", ""
	}

	h2 := findFirstElement(panelTitle, "h2")
	if h2 == nil {
		return "", ""
	}

	var timeStr string
	for c := h2.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			t := strings.TrimSpace(c.Data)
			if t != "" {
				timeStr = t
				break
			}
		}
	}

	meridiemNode := findByClass(h2, "meridiem")
	meridiem := ""
	if meridiemNode != nil {
		meridiem = strings.TrimSpace(extractText(meridiemNode))
	}

	return timeStr, meridiem
}

func extractSubTitle(panel *html.Node) string {
	panelTitle := findByClass(panel, "panel-title")
	if panelTitle == nil {
		return ""
	}

	divs := findAllElements(panelTitle, "div")
	for _, d := range divs {
		innerDivs := findAllElements(d, "div")
		for _, inner := range innerDivs {
			ps := findAllElements(inner, "p")
			for _, p := range ps {
				text := strings.TrimSpace(extractText(p))
				text = collapseSpaces(text)
				if text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func extractDescription(panel *html.Node) string {
	panelBody := findByClass(panel, "panel-body")
	if panelBody == nil {
		return ""
	}

	ps := findAllElements(panelBody, "p")
	if len(ps) >= 2 {
		return strings.TrimSpace(extractText(ps[1]))
	}
	return ""
}

func extractImage(panel *html.Node) string {
	thumb := findByClass(panel, "video-thumbnail")
	if thumb == nil {
		return ""
	}
	img := findFirstElement(thumb, "img")
	if img == nil {
		return ""
	}
	return getAttr(img, "src")
}

func collapseSpaces(s string) string {
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return s
}

func extractByClass(n *html.Node, class string) string {
	node := findByClass(n, class)
	if node == nil {
		return ""
	}
	return strings.TrimSpace(extractText(node))
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

func findFirstElement(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findFirstElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func findAllElements(n *html.Node, tag string) []*html.Node {
	var result []*html.Node
	if n.Type == html.ElementNode && n.Data == tag {
		result = append(result, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, findAllElements(c, tag)...)
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
