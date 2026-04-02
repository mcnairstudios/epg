package zuragtmn

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

var ulaanbaatar *time.Location

func init() {
	ulaanbaatar, _ = time.LoadLocation("Asia/Ulaanbaatar")
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

func (p *provider) Name() string        { return "zuragt.mn" }
func (p *provider) Description() string { return "Zuragt.mn Mongolian TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "zuragt.mn")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://m.zuragt.mn/channel/%s/?date=%s",
		channel.SiteID, date.Format("2006-01-02"))

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
		return nil, fmt.Errorf("HTTP %d from zuragtmn", resp.StatusCode)
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

	items := findListItems(doc)
	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	var programs []domain.Program
	curDate := date

	for _, li := range items {
		timeStr := extractTextByClass(li, "time")
		title := extractTextByClass(li, "program")
		if timeStr == "" || title == "" {
			continue
		}

		start := parseTime(strings.TrimSpace(timeStr), curDate)

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				curDate = curDate.Add(24 * time.Hour)
				start = parseTime(strings.TrimSpace(timeStr), curDate)
			}
			prev.Stop = start
		}

		stop := start.Add(30 * time.Minute)

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "zuragt.mn",
			Title:   strings.TrimSpace(title),
			Lang:    channel.Lang,
			Start:   start,
			Stop:    stop,
		})
	}

	return programs, nil
}

func parseTime(timeStr string, date time.Time) time.Time {
	t := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, ulaanbaatar)
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return t
	}
	var h, m int
	fmt.Sscanf(parts[0], "%d", &h)
	fmt.Sscanf(parts[1], "%d", &m)
	return time.Date(date.Year(), date.Month(), date.Day(), h, m, 0, 0, ulaanbaatar).UTC()
}

func findListItems(n *html.Node) []*html.Node {
	ul := findByClass(n, "programs-container")
	if ul == nil {
		var lis []*html.Node
		collectLIs(n, &lis)
		return lis
	}
	var lis []*html.Node
	for c := ul.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "li" {
			lis = append(lis, c)
		}
	}
	return lis
}

func collectLIs(n *html.Node, result *[]*html.Node) {
	if n.Type == html.ElementNode && n.Data == "li" {
		*result = append(*result, n)
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectLIs(c, result)
	}
}

func extractTextByClass(n *html.Node, class string) string {
	node := findByClass(n, class)
	if node == nil {
		return ""
	}
	return extractText(node)
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

func (p *provider) Timezone() string { return "Asia/Ulaanbaatar" }
