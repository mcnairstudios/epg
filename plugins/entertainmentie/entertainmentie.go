package entertainmentie

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
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

func (p *provider) Name() string        { return "entertainment.ie" }
func (p *provider) Description() string { return "Entertainment.ie Irish TV listings" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "entertainment.ie")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Format("02-01-2006")
	url := fmt.Sprintf("https://entertainment.ie/tv/%s/?date=%s&time=all-day", channel.SiteID, d)

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
		return nil, fmt.Errorf("HTTP %d from entertainmentie", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

type listingItem struct {
	title       string
	description string
	genres      string
	image       string
	timeStr     string
	duration    int
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	infoList := findByClass(doc, "info-list")
	if infoList == nil {
		return []domain.Program{}, nil
	}

	lis := findAllNodes(infoList, "li")
	var programs []domain.Program

	for _, li := range lis {
		item := extractListingItem(li)
		if item.timeStr == "" {
			continue
		}

		start := parseTimeUTC(item.timeStr, date)
		if start.IsZero() {
			continue
		}

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				start = start.AddDate(0, 0, 1)
			}
		}

		stop := start.Add(time.Duration(item.duration) * time.Minute)

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "entertainment.ie",
			Title:       item.title,
			Description: item.description,
			Image:       item.image,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		}

		if item.genres != "" {
			p.Category = item.genres
		}

		programs = append(programs, p)
	}

	if programs == nil {
		return []domain.Program{}, nil
	}
	return programs, nil
}

func extractListingItem(li *html.Node) listingItem {
	var item listingItem

	titleNode := findH3Text(li)
	item.title = strings.TrimSpace(titleNode)

	btnShare := findByClassAndTag(li, "btn-share", "a")
	if btnShare == nil {
		return item
	}

	item.description = getAttr(btnShare, "data-description")
	item.genres = getAttr(btnShare, "data-genres")
	item.image = getAttr(btnShare, "data-img")

	timeData := getAttr(btnShare, "data-time")
	if timeData != "" {
		parts := strings.SplitN(timeData, ", ", 2)
		if len(parts) == 2 {
			item.timeStr = parts[1]
		}
	}

	durStr := getAttr(btnShare, "data-duration")
	if durStr != "" {
		item.duration, _ = strconv.Atoi(strings.TrimSpace(durStr))
	}

	return item
}

func parseTimeUTC(timeStr string, date time.Time) time.Time {
	parts := strings.Split(strings.TrimSpace(timeStr), ":")
	if len(parts) != 2 {
		return time.Time{}
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}
	}

	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, time.UTC)
}

func findH3Text(n *html.Node) string {
	h3 := findNode(n, "h3")
	if h3 == nil {
		return ""
	}
	return extractText(h3)
}

func findByClassAndTag(n *html.Node, class, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		for _, a := range n.Attr {
			if a.Key == "class" && strings.Contains(a.Val, class) {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findByClassAndTag(c, class, tag); found != nil {
			return found
		}
	}
	return nil
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

func findAllNodes(n *html.Node, tag string) []*html.Node {
	var result []*html.Node
	if n.Type == html.ElementNode && n.Data == tag {
		result = append(result, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, findAllNodes(c, tag)...)
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
