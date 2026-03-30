package passienl

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
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{client: httpclient.Browser()})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "passie.nl" }
func (p *provider) Description() string { return "Passie.nl TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "passie.nl")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()
	url := fmt.Sprintf("https://passie.nl/tvgids.php?day=%s", d.Format("2006-01-02"))

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
		return nil, fmt.Errorf("HTTP %d from passienl", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel, date)
}

var timeRe = regexp.MustCompile(`(\d{1,2}:\d{2})\s*CET`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation("CET")
	if err != nil {
		return nil, err
	}

	guideFocus := findByID(doc, "guide-focus")
	if guideFocus == nil {
		return []domain.Program{}, nil
	}

	items := findItemWrappers(guideFocus)
	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	currentDate := date.AddDate(0, 0, -1)
	programs := make([]domain.Program, 0, len(items))

	for _, item := range items {
		title, timeStr := extractItemData(item)
		if title == "" || timeStr == "" {
			continue
		}

		matches := timeRe.FindStringSubmatch(timeStr)
		if matches == nil {
			continue
		}

		fullTime := fmt.Sprintf("%s %s", currentDate.Format("2006-01-02"), matches[1])
		start, err := time.ParseInLocation("2006-01-02 15:04", fullTime, loc)
		if err != nil {
			continue
		}

		if len(programs) > 0 {
			if start.Before(programs[len(programs)-1].Start) {
				start = start.Add(24 * time.Hour)
				currentDate = currentDate.AddDate(0, 0, 1)
			}
			programs[len(programs)-1].Stop = start.UTC()
		} else {
			if start.Hour() < 23 {
				start = start.Add(24 * time.Hour)
				currentDate = currentDate.AddDate(0, 0, 1)
			}
		}

		minute := start.Minute()
		stopMinute := 60 - minute
		stop := start.Add(time.Duration(stopMinute) * time.Minute)

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "passie.nl",
			Title:   title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		}
		programs = append(programs, p)
	}

	return programs, nil
}

func findByID(doc *html.Node, id string) *html.Node {
	var result *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if result != nil {
			return
		}
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				if attr.Key == "id" && attr.Val == id {
					result = n
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return result
}

func findItemWrappers(root *html.Node) []*html.Node {
	var items []*html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "item-wrapper") {
			if hasTimeContent(n) {
				items = append(items, n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(root)
	return items
}

func hasTimeContent(n *html.Node) bool {
	text := textContent(n)
	return timeRe.MatchString(text)
}

func extractItemData(item *html.Node) (title, timeStr string) {
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			classes := getClasses(n)
			if containsAll(classes, "col-xs-6", "col-sm-7") {
				bText := findBoldText(n)
				if bText != "" {
					title = strings.TrimSpace(bText)
				}
			}
			if containsAll(classes, "col-xs-3", "col-sm-2") && !containsAll(classes, "hide") {
				text := strings.TrimSpace(textContent(n))
				if timeRe.MatchString(text) {
					timeStr = text
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(item)
	return
}

func findBoldText(n *html.Node) string {
	var result string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if result != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "b" {
			result = strings.TrimSpace(textContent(n))
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return result
}

func getClasses(n *html.Node) []string {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			return strings.Fields(attr.Val)
		}
	}
	return nil
}

func containsAll(classes []string, targets ...string) bool {
	for _, target := range targets {
		found := false
		for _, c := range classes {
			if c == target {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func hasClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			for _, c := range strings.Fields(attr.Val) {
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
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textContent(c))
	}
	return sb.String()
}
