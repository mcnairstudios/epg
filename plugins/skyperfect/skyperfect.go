package skyperfect

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

var tokyoTZ *time.Location

//go:embed channels.json
var channelsData []byte

func init() {
	tokyoTZ, _ = time.LoadLocation("Asia/Tokyo")
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

func (p *provider) Name() string        { return "skyperfectv.co.jp" }
func (p *provider) Description() string { return "SKY PerfecTV! Japanese satellite TV" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "skyperfectv.co.jp")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	parts := strings.SplitN(channel.SiteID, "_", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid site_id format: %s", channel.SiteID)
	}
	serviceType := parts[0]
	code := parts[1]

	url := fmt.Sprintf(
		"https://www.skyperfectv.co.jp/program/schedule/%s/channel:%s/date:%s",
		serviceType, code, date.Format("060102"),
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", "adult_auth=true")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from skyperfect", resp.StatusCode)
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
		return nil, err
	}

	type sectionDef struct {
		id       string
		addition int
	}

	sections := []sectionDef{
		{id: "js-am", addition: 0},
		{id: "js-pm", addition: 0},
		{id: "js-md", addition: 1},
	}

	var programs []domain.Program

	for _, sec := range sections {
		tr := findNodeByID(doc, sec.id)
		if tr == nil {
			continue
		}

		tds := findDirectChildrenByTag(tr, "td")
		for colIdx, td := range tds {
			today := date.Add(time.Duration(colIdx+sec.addition) * 24 * time.Hour)
			todayInTZ := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, tokyoTZ)

			items := findNodesByClass(td, "p-program__item")
			for _, item := range items {
				rangeNode := findNodeByClass(item, "p-program__range")
				if rangeNode == nil {
					continue
				}

				rangeText := textContent(rangeNode)
				timeParts := strings.SplitN(rangeText, "\u301c", 2)
				if len(timeParts) != 2 {
					continue
				}

				startTime := parseTimeOfDay(strings.TrimSpace(timeParts[0]), todayInTZ)
				stopTime := parseTimeOfDay(strings.TrimSpace(timeParts[1]), todayInTZ)

				nameNode := findNodeByClass(item, "p-program__name")
				title := ""
				if nameNode != nil {
					title = strings.TrimSpace(textContent(nameNode))
				}

				var image string
				thumbNode := findNodeByClass(item, "js-program_thumbnail")
				if thumbNode != nil {
					image = getAttr(thumbNode, "data-lazysrc")
				}

				programs = append(programs, domain.Program{
					Channel: channel.XmltvID,
					Source:  "skyperfectv.co.jp",
					Title:   title,
					Image:   image,
					Lang:    channel.Lang,
					Start:   startTime.UTC(),
					Stop:    stopTime.UTC(),
				})
			}
		}
	}

	if programs == nil {
		programs = []domain.Program{}
	}

	return programs, nil
}

func parseTimeOfDay(s string, day time.Time) time.Time {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return day
	}
	hour := 0
	minute := 0
	fmt.Sscanf(parts[0], "%d", &hour)
	fmt.Sscanf(parts[1], "%d", &minute)
	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())
}

func findNodeByID(n *html.Node, id string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == id {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findNodeByID(c, id); found != nil {
			return found
		}
	}
	return nil
}

func findDirectChildrenByTag(n *html.Node, tag string) []*html.Node {
	var nodes []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			nodes = append(nodes, c)
		}
	}
	return nodes
}

func findNodesByClass(n *html.Node, class string) []*html.Node {
	var nodes []*html.Node
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && strings.Contains(a.Val, class) {
				nodes = append(nodes, n)
				break
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		nodes = append(nodes, findNodesByClass(c, class)...)
	}
	return nodes
}

func findNodeByClass(n *html.Node, class string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && strings.Contains(a.Val, class) {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findNodeByClass(c, class); found != nil {
			return found
		}
	}
	return nil
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
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
