package tvgidsnl

import (
	"fmt"
	"io"
	"net/http"
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

var amsterdam *time.Location

func init() {
	amsterdam, _ = time.LoadLocation("Europe/Amsterdam")
	plugin.Register(&provider{
		client: httpclient.Browser(),
	})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "tvgids.nl" }
func (p *provider) Description() string { return "TVgids.nl Dutch TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvgids.nl")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Format("02-01-2006")
	url := fmt.Sprintf("https://www.tvgids.nl/gids/%s/%s", d, channel.SiteID)

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
		return nil, fmt.Errorf("HTTP %d from tvgidsnl", resp.StatusCode)
	}

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

	guideDiv := findByClass(doc, "guide__guide")
	if guideDiv == nil {
		return []domain.Program{}, nil
	}

	items := findAllByClass(guideDiv, "program")
	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	prevDay := date.AddDate(0, 0, -1)
	currentDate := prevDay
	var programs []domain.Program

	for _, item := range items {
		title := extractTextByClass(item, "program__title")
		description := extractTextByClass(item, "program__text")
		image := extractDataSrcByClass(item, "program__thumbnail")
		timeStr := extractStartTime(item)

		if timeStr == "" || title == "" {
			continue
		}

		start := parseTime(timeStr, currentDate)

		if len(programs) > 0 {
			prev := programs[len(programs)-1]
			if start.Before(prev.Start) {
				currentDate = currentDate.AddDate(0, 0, 1)
				start = parseTime(timeStr, currentDate)
			}
			programs[len(programs)-1].Stop = start
		}

		stop := start.Add(30 * time.Minute)

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tvgids.nl",
			Title:       title,
			Description: description,
			Image:       image,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

func parseTime(timeStr string, date time.Time) time.Time {
	parts := strings.Split(strings.TrimSpace(timeStr), ":")
	if len(parts) != 2 {
		return time.Time{}
	}
	hour, minute := 0, 0
	fmt.Sscanf(parts[0], "%d", &hour)
	fmt.Sscanf(parts[1], "%d", &minute)

	t := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, amsterdam)
	return t.UTC()
}

func extractStartTime(n *html.Node) string {
	node := findByClass(n, "program__starttime")
	if node == nil {
		return ""
	}
	var result string
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			result += c.Data
		}
	}
	return strings.TrimSpace(result)
}

func extractTextByClass(n *html.Node, class string) string {
	node := findByClass(n, class)
	if node == nil {
		return ""
	}
	return strings.TrimSpace(extractText(node))
}

func extractDataSrcByClass(n *html.Node, class string) string {
	node := findByClass(n, class)
	if node == nil {
		return ""
	}
	return getAttr(node, "data-src")
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

func findAllByClass(n *html.Node, class string) []*html.Node {
	var result []*html.Node
	if n.Type == html.ElementNode {
		cls := getAttr(n, "class")
		for _, c := range strings.Fields(cls) {
			if c == class {
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
