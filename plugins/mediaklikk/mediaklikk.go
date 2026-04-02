package mediaklikk

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func (p *provider) Name() string        { return "mediaklikk.hu" }
func (p *provider) Description() string { return "Mediaklikk Hungary EPG" }

const endpoint = "https://mediaklikk.hu/wp-content/plugins/hms-global-widgets/widgets/programGuide/programGuideInterface.php"

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "mediaklikk.hu")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	form := url.Values{}
	form.Set("ChannelIds", channel.SiteID+",")
	form.Set("Date", date.Format("2006-01-02"))

	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from mediaklikk", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

var bgImageRe = regexp.MustCompile(`url\('([^']+)'\)`)

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return []domain.Program{}, nil
	}

	items := findProgramBodies(doc)
	programs := make([]domain.Program, 0, len(items))

	for _, item := range items {
		from := getAttr(item, "data-from")
		till := getAttr(item, "data-till")

		start := parseTime(from)
		if start.IsZero() {
			continue
		}

		stop := parseTime(till)
		if stop.IsZero() {
			stop = start.Add(30 * time.Minute)
		}

		title := extractTitle(item)
		desc := extractDescription(item)
		image := extractImage(item)

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "mediaklikk.hu",
			Title:       title,
			Description: desc,
			Image:       image,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		})
	}

	return programs, nil
}

func parseTime(s string) time.Time {
	if s == "" || strings.HasPrefix(s, "+") {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02 15:04:05-0700", s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func findProgramBodies(n *html.Node) []*html.Node {
	var results []*html.Node
	if n.Type == html.ElementNode && n.Data == "li" && hasClass(n, "program_body") {
		results = append(results, n)
		return results
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		results = append(results, findProgramBodies(c)...)
	}
	return results
}

func hasClass(n *html.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range strings.Fields(a.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func extractTitle(n *html.Node) string {
	h1 := findElement(n, func(node *html.Node) bool {
		if node.Type != html.ElementNode || node.Data != "h1" {
			return false
		}
		parent := node.Parent
		return parent != nil && parent.Data == "div" && hasClass(parent, "program_info")
	})
	if h1 != nil {
		return strings.TrimSpace(textContent(h1))
	}
	return ""
}

func extractDescription(n *html.Node) string {
	p := findElement(n, func(node *html.Node) bool {
		if node.Type != html.ElementNode || node.Data != "p" {
			return false
		}
		parent := node.Parent
		if parent == nil || parent.Data != "div" || !hasClass(parent, "program_description") {
			return false
		}
		grandParent := parent.Parent
		return grandParent != nil && grandParent.Data == "div" && hasClass(grandParent, "program_about")
	})
	if p != nil {
		return strings.TrimSpace(textContent(p))
	}
	return ""
}

func extractImage(n *html.Node) string {
	div := findElement(n, func(node *html.Node) bool {
		if node.Type != html.ElementNode || node.Data != "div" || !hasClass(node, "program_photo") {
			return false
		}
		parent := node.Parent
		return parent != nil && parent.Data == "div" && hasClass(parent, "program_about")
	})
	if div != nil {
		style := getAttr(div, "style")
		m := bgImageRe.FindStringSubmatch(style)
		if m != nil {
			imageURL := m[1]
			if strings.HasPrefix(imageURL, "//") {
				return fmt.Sprintf("https:%s", imageURL)
			}
			return imageURL
		}
	}
	return ""
}

func findElement(n *html.Node, match func(*html.Node) bool) *html.Node {
	if match(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findElement(c, match); result != nil {
			return result
		}
	}
	return nil
}

func textContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var buf strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		buf.WriteString(textContent(c))
	}
	return buf.String()
}

func (p *provider) Timezone() string { return "Europe/Budapest" }
