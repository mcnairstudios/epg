package berrymedia

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

var seoulTZ *time.Location

//go:embed channels.json
var channelsData []byte

func init() {
	seoulTZ, _ = time.LoadLocation("Asia/Seoul")
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

func (p *provider) Name() string        { return "berrymedia.co.kr" }
func (p *provider) Description() string { return "BerryMedia Korean TV channels" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "berrymedia.co.kr")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	apiURL := fmt.Sprintf("http://www.berrymedia.co.kr/schedule_proc%s.php", channel.SiteID)

	weekStart := startOfWeek(date)
	weekEnd := weekStart.AddDate(0, 0, 6)
	form := url.Values{}
	form.Set("week", fmt.Sprintf("%s~%s", weekStart.Format("2006-01-02"), weekEnd.Format("2006-01-02")))
	form.Set("day", date.Format("2006-01-02"))

	req, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from berrymedia", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

func startOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return t.AddDate(0, 0, -(weekday - 1))
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	items := findDDElements(doc)
	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	var programs []domain.Program
	for _, dd := range items {
		spans := findDirectChildren(dd, "span")
		if len(spans) == 0 {
			continue
		}

		timeStr := strings.TrimSpace(extractDirectText(spans[0]))
		if timeStr == "" || len(timeStr) != 5 {
			continue
		}

		start, err := time.ParseInLocation("2006-01-02 15:04", date.Format("2006-01-02")+" "+timeStr, seoulTZ)
		if err != nil {
			continue
		}

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				date = date.AddDate(0, 0, 1)
				start, _ = time.ParseInLocation("2006-01-02 15:04", date.Format("2006-01-02")+" "+timeStr, seoulTZ)
			}
			prev.Stop = start.UTC()
		}

		title := ""
		category := ""
		rating := ""

		for i, span := range spans {
			if hasClass(span, "sdfsdf") {
				title = strings.TrimSpace(extractDirectText(span))
			}
			if i == 1 {
				p := findFirstChild(span, "p")
				if p != nil {
					category = strings.TrimSpace(extractText(p))
				}
			}
			if i == 4 {
				p := findFirstChild(span, "p")
				if p != nil {
					rating = strings.TrimSpace(extractText(p))
				}
			}
		}

		stop := start.Add(30 * time.Minute)

		prog := domain.Program{
			Channel:  channel.XmltvID,
			Source:   "berrymedia.co.kr",
			Title:    title,
			Category: category,
			Lang:     channel.Lang,
			Start:    start.UTC(),
			Stop:     stop.UTC(),
		}

		if rating != "" {
			prog.Description = fmt.Sprintf("KMRB:%s", rating)
		}

		programs = append(programs, prog)
	}

	if programs == nil {
		return []domain.Program{}, nil
	}
	return programs, nil
}

func findDDElements(n *html.Node) []*html.Node {
	var scTime *html.Node
	var findSCTime func(*html.Node)
	findSCTime = func(node *html.Node) {
		if node.Type == html.ElementNode {
			cls := getAttr(node, "class")
			for _, c := range strings.Fields(cls) {
				if c == "sc_time" {
					scTime = node
					return
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			findSCTime(c)
			if scTime != nil {
				return
			}
		}
	}
	findSCTime(n)

	if scTime == nil {
		return nil
	}

	var dds []*html.Node
	for c := scTime.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "dd" {
			dds = append(dds, c)
		}
	}
	return dds
}

func findDirectChildren(n *html.Node, tag string) []*html.Node {
	var result []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			result = append(result, c)
		}
	}
	return result
}

func findFirstChild(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			return c
		}
	}
	return nil
}

func hasClass(n *html.Node, class string) bool {
	cls := getAttr(n, "class")
	for _, c := range strings.Fields(cls) {
		if c == class {
			return true
		}
	}
	return false
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

func (p *provider) Timezone() string { return "Asia/Seoul" }
