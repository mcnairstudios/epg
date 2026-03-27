package tv24se

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"golang.org/x/net/html"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
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

func (p *provider) Name() string        { return "tv24.se" }
func (p *provider) Description() string { return "TV24.se Nordic TV guide (326 channels)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tv24.se")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	day := date.Truncate(24 * time.Hour).UTC()
	url := fmt.Sprintf("https://tv24.se/x/channel/%s/0/%s", channel.SiteID, day.Format("2006-01-02"))

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

	items := Parse(body, day)

	for i := range items {
		detailURL := fmt.Sprintf("https://tv24.se/x%s/0/0", items[i].href)
		details, err := p.fetchDetails(detailURL)
		if err != nil {
			continue
		}
		applyDetails(&items[i], details)
	}

	programs := make([]domain.Program, len(items))
	for i, item := range items {
		programs[i] = domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tv24.se",
			Title:       item.Title,
			SubTitle:    item.SubTitle,
			Description: item.Description,
			Category:    item.Category,
			Season:      item.Season,
			Episode:     item.Episode,
			Image:       item.Image,
			Lang:        channel.Lang,
			Start:       item.Start,
			Stop:        item.Stop,
		}
	}

	return programs, nil
}

func (p *provider) fetchDetails(url string) (*detailResponse, error) {
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

	var detail detailResponse
	if err := json.Unmarshal(body, &detail); err != nil {
		return nil, err
	}

	return &detail, nil
}

type ProgramItem struct {
	href        string
	Title       string
	SubTitle    string
	Description string
	Category    string
	Season      int
	Episode     int
	Image       string
	Actors      []string
	Start       time.Time
	Stop        time.Time
}

type detailResponse struct {
	ContentBefore string `json:"contentBefore"`
	ContentAfter  string `json:"contentAfter"`
}

func Parse(data []byte, date time.Time) []ProgramItem {
	content := string(data)
	if strings.TrimSpace(content) == "" {
		return nil
	}

	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil
	}

	var items []ProgramItem
	var currentDate = date

	programs := findElementsByClass(doc, "a", "program")
	for _, prog := range programs {
		timeText := getTextByClass(prog, "span", "time")
		title := getTextByTag(prog, "h3")
		href := getAttr(prog, "href")

		start, err := time.Parse("2006-01-02 15:04", currentDate.Format("2006-01-02")+" "+timeText)
		if err != nil {
			continue
		}
		start = start.UTC()

		if len(items) > 0 {
			prev := &items[len(items)-1]
			if start.Before(prev.Start) {
				currentDate = currentDate.Add(24 * time.Hour)
				start, err = time.Parse("2006-01-02 15:04", currentDate.Format("2006-01-02")+" "+timeText)
				if err != nil {
					continue
				}
				start = start.UTC()
			}
			prev.Stop = start
		}

		stop := start.Add(30 * time.Minute)

		items = append(items, ProgramItem{
			href:  href,
			Title: title,
			Start: start,
			Stop:  stop,
		})
	}

	return items
}

func ParseDetails(data []byte) *ProgramItem {
	var detail detailResponse
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil
	}

	item := &ProgramItem{}
	applyDetails(item, &detail)
	return item
}

var (
	seasonRe  = regexp.MustCompile(`Säsong (\d+)`)
	episodeRe = regexp.MustCompile(`Avsnitt (\d+)`)
	imageRe   = regexp.MustCompile(`background-image: url\('([^']+)'\)`)
)

func applyDetails(item *ProgramItem, detail *detailResponse) {
	combined := detail.ContentBefore + detail.ContentAfter
	if combined == "" {
		return
	}

	doc, err := html.Parse(strings.NewReader(combined))
	if err != nil {
		return
	}

	item.Image = parseImage(doc)
	item.Description = parseDescription(doc)
	item.Actors = parseActors(doc)
	item.Category = parseCategory(doc)

	subTitleText := getTextByClass(doc, "span", "sub-title")
	if subTitleText != "" {
		parts := strings.SplitN(subTitleText, ": ", 2)
		if len(parts) == 2 {
			item.SubTitle = parts[1]
		}

		if m := seasonRe.FindStringSubmatch(subTitleText); len(m) == 2 {
			item.Season, _ = strconv.Atoi(m[1])
		}
		if m := episodeRe.FindStringSubmatch(subTitleText); len(m) == 2 {
			item.Episode, _ = strconv.Atoi(m[1])
		}
	}
}

func parseImage(doc *html.Node) string {
	nodes := findElementsByClass(doc, "div", "actual")
	for _, n := range nodes {
		style := getAttr(n, "style")
		if m := imageRe.FindStringSubmatch(style); len(m) == 2 {
			return m[1]
		}
	}
	return ""
}

func parseDescription(doc *html.Node) string {
	infoNodes := findElementsByClass(doc, "div", "info")
	for _, info := range infoNodes {
		pNodes := findElementsByTag(info, "p")
		for _, p := range pNodes {
			text := strings.TrimSpace(getNodeText(p))
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func parseActors(doc *html.Node) []string {
	var actors []string
	nameNodes := findElementsByClass(doc, "span", "name")
	for _, n := range nameNodes {
		name := strings.TrimSpace(getNodeText(n))
		if name != "" {
			actors = append(actors, name)
		}
	}
	return actors
}

func parseCategory(doc *html.Node) string {
	dtNodes := findElementsByTag(doc, "dt")
	for _, dt := range dtNodes {
		text := strings.TrimSpace(getNodeText(dt))
		if text == "Kategori" {
			dd := nextSiblingElement(dt)
			if dd != nil {
				return strings.TrimSpace(getNodeText(dd))
			}
		}
	}
	return ""
}

func findElementsByClass(n *html.Node, tag, class string) []*html.Node {
	var results []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == tag {
			for _, attr := range n.Attr {
				if attr.Key == "class" && containsClass(attr.Val, class) {
					results = append(results, n)
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return results
}

func findElementsByTag(n *html.Node, tag string) []*html.Node {
	var results []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == tag {
			results = append(results, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return results
}

func containsClass(classes, target string) bool {
	for _, c := range strings.Fields(classes) {
		if c == target {
			return true
		}
	}
	return false
}

func getTextByClass(root *html.Node, tag, class string) string {
	nodes := findElementsByClass(root, tag, class)
	if len(nodes) == 0 {
		return ""
	}
	return strings.TrimSpace(getNodeText(nodes[0]))
}

func getTextByTag(root *html.Node, tag string) string {
	nodes := findElementsByTag(root, tag)
	if len(nodes) == 0 {
		return ""
	}
	return strings.TrimSpace(getNodeText(nodes[0]))
}

func getNodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(getNodeText(c))
	}
	return sb.String()
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func nextSiblingElement(n *html.Node) *html.Node {
	for s := n.NextSibling; s != nil; s = s.NextSibling {
		if s.Type == html.ElementNode {
			return s
		}
	}
	return nil
}

func NewWithClient(client *http.Client) *provider {
	return &provider{client: client}
}

type TestableProvider struct{}

func (tp *TestableProvider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tv24.se")
}
