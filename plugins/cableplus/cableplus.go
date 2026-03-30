package cableplus

import (
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

var montevideo *time.Location

func init() {
	montevideo = time.FixedZone("UYT", -3*60*60)
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

func (p *provider) Name() string        { return "cableplus.com.uy" }
func (p *provider) Description() string { return "Cable Plus Uruguay (171 channels)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "cableplus.com.uy")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	params := url.Values{}
	params.Set("idAlineacion", "3017")
	params.Set("idSenial", channel.SiteID)
	params.Set("fecha", date.Format("2006-01-02"))
	params.Set("hora", "00:00")

	req, err := http.NewRequest(http.MethodPost, "https://www.reportv.com.ar/finder/channel", strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from cableplus", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	dateStr := fmt.Sprintf("%02d-%02d", date.Day(), date.Month())
	dateStr2 := fmt.Sprintf("%02d/%02d", date.Day(), date.Month())
	year := date.Format("2006")

	items := parseFeaturedItems(doc, dateStr, year)
	items = append(items, parseCarouselItems(doc, dateStr2, year)...)

	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	var programs []domain.Program
	for i, item := range items {
		prev := len(programs) - 1
		if prev >= 0 {
			programs[prev].Stop = item.start
		}
		p := domain.Program{
			Channel:  channel.XmltvID,
			Source:   "cableplus.com.uy",
			Title:    item.title,
			Category: item.category,
			Image:    item.image,
			Lang:     channel.Lang,
			Start:    item.start,
			Stop:     item.start.Add(30 * time.Minute),
		}
		programs = append(programs, p)
		_ = i
	}

	return programs, nil
}

type parsedItem struct {
	title    string
	category string
	image    string
	start    time.Time
}

var gridDateTimeRE = regexp.MustCompile(`(\d{2}-\d{2})\s+(\d{2}:\d{2})hs\.`)
var carouselDateTimeRE = regexp.MustCompile(`(\d{2}/\d{2})\s+(\d{2}:\d{2})hs\.`)

func parseFeaturedItems(doc *html.Node, dateStr string, year string) []parsedItem {
	var items []parsedItem

	vistaPc := findByClass(doc, "vista-pc")
	if vistaPc == nil {
		return items
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && hasClass(n, "channel-programa") {
			timeText := extractTextByClass(n, "grid_fecha_hora")
			if strings.Contains(timeText, dateStr) {
				matches := gridDateTimeRE.FindStringSubmatch(timeText)
				if len(matches) == 3 {
					t := parseGridTime(year, matches[1], matches[2])
					title := extractProgramTitle(n)
					image := extractImage(n)
					category := extractCategory(n)
					items = append(items, parsedItem{
						title:    title,
						category: category,
						image:    image,
						start:    t,
					})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(vistaPc)

	return items
}

func parseCarouselItems(doc *html.Node, dateStr string, year string) []parsedItem {
	var items []parsedItem

	owlPc := findByID(doc, "owl-pc")
	if owlPc == nil {
		return items
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && hasClass(n, "item-program") {
			timeText := extractTextByClass(n, "fechaHora")
			if strings.Contains(timeText, dateStr) {
				matches := carouselDateTimeRE.FindStringSubmatch(timeText)
				if len(matches) == 3 {
					t := parseCarouselTime(year, matches[1], matches[2])
					title := extractCarouselTitle(n)
					image := extractImage(n)
					category := extractCategory(n)
					items = append(items, parsedItem{
						title:    title,
						category: category,
						image:    image,
						start:    t,
					})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(owlPc)

	return items
}

func parseGridTime(year, dateStr, timeStr string) time.Time {
	parts := strings.Split(dateStr, "-")
	if len(parts) != 2 {
		return time.Time{}
	}
	s := fmt.Sprintf("%s-%s-%s %s", year, parts[1], parts[0], timeStr)
	t, _ := time.ParseInLocation("2006-01-02 15:04", s, montevideo)
	return t.UTC()
}

func parseCarouselTime(year, dateStr, timeStr string) time.Time {
	parts := strings.Split(dateStr, "/")
	if len(parts) != 2 {
		return time.Time{}
	}
	s := fmt.Sprintf("%s-%s-%s %s", year, parts[1], parts[0], timeStr)
	t, _ := time.ParseInLocation("2006-01-02 15:04", s, montevideo)
	return t.UTC()
}

func extractProgramTitle(n *html.Node) string {
	span := findNodeByTag(findByClass(n, "programa-titulo"), "span")
	if span != nil {
		return strings.TrimSpace(extractText(span))
	}
	el := findByClass(n, "evento_titulo")
	if el != nil {
		return strings.TrimSpace(extractText(el))
	}
	return ""
}

func extractCarouselTitle(n *html.Node) string {
	el := findByClassExact(n, "evento_titulo texto_a_continuacion dotdotdot")
	if el == nil {
		el = findByClass(n, "evento_titulo")
	}
	if el != nil {
		return strings.TrimSpace(extractText(el))
	}
	return ""
}

func extractCategory(n *html.Node) string {
	el := findByClass(n, "evento_genero")
	if el == nil {
		return ""
	}
	return strings.TrimSpace(extractText(el))
}

func extractImage(n *html.Node) string {
	imgs := findAllNodesByTag(n, "img")
	for _, img := range imgs {
		src := getAttr(img, "data-src")
		if src == "" {
			src = getAttr(img, "src")
		}
		if src != "" && !strings.Contains(src, "Logos") {
			return src
		}
	}
	return ""
}

func extractTextByClass(n *html.Node, class string) string {
	el := findByClass(n, class)
	if el == nil {
		return ""
	}
	return strings.TrimSpace(extractText(el))
}

func findByClass(n *html.Node, class string) *html.Node {
	if n == nil {
		return nil
	}
	if n.Type == html.ElementNode {
		cls := getAttr(n, "class")
		if strings.Contains(cls, class) {
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

func findByClassExact(n *html.Node, class string) *html.Node {
	if n == nil {
		return nil
	}
	if n.Type == html.ElementNode {
		cls := getAttr(n, "class")
		if cls == class {
			return n
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findByClassExact(c, class); found != nil {
			return found
		}
	}
	return nil
}

func findByID(n *html.Node, id string) *html.Node {
	if n == nil {
		return nil
	}
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

func hasClass(n *html.Node, class string) bool {
	cls := getAttr(n, "class")
	return strings.Contains(cls, class)
}

func findNodeByTag(n *html.Node, tag string) *html.Node {
	if n == nil {
		return nil
	}
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findNodeByTag(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func findAllNodesByTag(n *html.Node, tag string) []*html.Node {
	var result []*html.Node
	if n == nil {
		return result
	}
	if n.Type == html.ElementNode && n.Data == tag {
		result = append(result, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, findAllNodesByTag(c, tag)...)
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
