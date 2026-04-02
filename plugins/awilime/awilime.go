package awilime

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

var budapestTZ, _ = time.LoadLocation("Europe/Budapest")

func init() {
	plugin.Register(&provider{client: httpclient.Browser()})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "awilime.com" }
func (p *provider) Description() string { return "Awilime Hungarian TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "awilime.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://www.awilime.com/tv/napi_musor/%s/%s",
		channel.SiteID,
		date.Format("2006_01_02"),
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from awilime", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

var timeRe = regexp.MustCompile(`^\d`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	body := findNodeByID(doc, "body")
	if body == nil {
		body = doc
	}

	tdcDiv := findNodeByClass(body, "tdc")
	if tdcDiv == nil {
		return []domain.Program{}, nil
	}

	td2Div := findNodeByClass(tdcDiv, "td2")
	if td2Div == nil {
		return []domain.Program{}, nil
	}

	itemDivs := findChildDivs(td2Div)
	if len(itemDivs) == 0 {
		return []domain.Program{}, nil
	}

	dateStr := date.Format("2006-01-02")
	programs := make([]domain.Program, 0, len(itemDivs))

	for _, itemDiv := range itemDivs {
		timeStr := parseBoldTimeText(itemDiv)
		if timeStr == "" || !timeRe.MatchString(timeStr) {
			continue
		}

		start, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+timeStr, budapestTZ)
		if err != nil {
			continue
		}

		if len(programs) > 0 {
			programs[len(programs)-1].Stop = start.UTC()
		}

		title := parseTitleFromBoldLink(itemDiv)
		subTitle := parseSubTitle(itemDiv)
		description := parseDescription(itemDiv)
		stop := start.Add(30 * time.Minute)

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "awilime.com",
			Title:       title,
			SubTitle:    subTitle,
			Description: description,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		})
	}

	return programs, nil
}

func parseBoldTimeText(n *html.Node) string {
	b := findElement(n, "b")
	if b == nil {
		return ""
	}
	var result string
	for c := b.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			result += c.Data
		}
	}
	return strings.TrimSpace(result)
}

func parseTitleFromBoldLink(n *html.Node) string {
	b := findElement(n, "b")
	if b == nil {
		return ""
	}
	a := findElement(b, "a")
	if a == nil {
		return ""
	}
	return strings.TrimSpace(textContent(a))
}

func parseSubTitle(n *html.Node) string {
	i := findElement(n, "i")
	if i == nil {
		return ""
	}
	var sb strings.Builder
	for c := i.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "s" {
			continue
		}
		sb.WriteString(textContent(c))
	}
	return strings.TrimSpace(sb.String())
}

func parseDescription(n *html.Node) string {
	p := findElement(n, "p")
	if p == nil {
		return ""
	}
	return strings.TrimSpace(textContent(p))
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

func findNodeByClass(n *html.Node, class string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && containsClass(a.Val, class) {
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

func findChildDivs(n *html.Node) []*html.Node {
	var divs []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "div" {
			divs = append(divs, c)
		}
	}
	return divs
}

func findElement(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func containsClass(classes string, target string) bool {
	for _, c := range strings.Fields(classes) {
		if c == target {
			return true
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

func (p *provider) Timezone() string { return "Europe/Budapest" }
