package elcinema

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
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

func (p *provider) Name() string        { return "elcinema.com" }
func (p *provider) Description() string { return "elCinema Arabic/English TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "elcinema.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	langPath := "/"
	if channel.Lang == "en" {
		langPath = "/en/"
	}

	url := fmt.Sprintf("https://elcinema.com%stvguide/%s/", langPath, channel.SiteID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

var timeRe = regexp.MustCompile(`(\d{1,2}):(\d{2})\s*(AM|PM|مساءً|صباحًا)`)
var durationRe = regexp.MustCompile(`(\d+)\s*(?:minutes|دقيقة)`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	dateStr := formatDateString(date, channel.Lang)
	section := findDateSection(doc, dateStr)
	if section == nil {
		return []domain.Program{}, nil
	}

	items := findAllByClass(section, "padded-half")
	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		title := extractTitle(item, channel.Lang)
		if title == "" {
			continue
		}

		timeStr := extractTimeText(item, channel.Lang)
		durStr := extractDurationText(item, channel.Lang)

		start := parseTime12h(timeStr, date)
		if start.IsZero() {
			continue
		}

		dur := parseDuration(durStr)
		stop := start.Add(time.Duration(dur) * time.Minute)

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "elcinema.com",
			Title:   title,
			Lang:    channel.Lang,
			Start:   start,
			Stop:    stop,
		}

		cat := extractCategory(item, channel.Lang)
		if cat != "" {
			p.Category = cat
		}

		img := extractImage(item)
		if img != "" {
			p.Image = img
		}

		desc := extractDescription(item)
		if desc != "" {
			p.Description = desc
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func formatDateString(date time.Time, lang string) string {
	dayName := date.Weekday().String()
	day := date.Day()

	if lang == "ar" {
		weekdays := map[string]string{
			"Sunday": "الأحد", "Monday": "الاثنين", "Tuesday": "الثلاثاء",
			"Wednesday": "الأربعاء", "Thursday": "الخميس", "Friday": "الجمعة", "Saturday": "السبت",
		}
		if ar, ok := weekdays[dayName]; ok {
			return fmt.Sprintf("%s %d", ar, day)
		}
	}

	return fmt.Sprintf("%s %d", dayName, day)
}

func findDateSection(n *html.Node, dateStr string) *html.Node {
	datesNodes := findAllByClass(n, "dates")
	for _, dn := range datesNodes {
		text := strings.TrimSpace(extractText(dn))
		text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
		if strings.Contains(text, dateStr) {
			parent := dn.Parent
			if parent == nil {
				continue
			}
			for sib := parent.NextSibling; sib != nil; sib = sib.NextSibling {
				if sib.Type == html.ElementNode {
					return sib
				}
			}
		}
	}
	return nil
}

func extractTitle(item *html.Node, lang string) string {
	var titleDiv *html.Node
	if lang == "en" || lang == "ar" {
		titleDiv = findByClassExact(item, "columns small-6 large-3")
		if titleDiv == nil {
			titleDiv = findByClassExact(item, "columns small-7 large-11")
		}
	}
	if titleDiv == nil {
		return ""
	}

	a := findNode(titleDiv, "a")
	if a != nil {
		return strings.TrimSpace(extractText(a))
	}

	lis := findAllNodes(titleDiv, "li")
	if len(lis) > 0 {
		return strings.TrimSpace(extractText(lis[0]))
	}

	return ""
}

func extractCategory(item *html.Node, lang string) string {
	titleDiv := findByClassExact(item, "columns small-6 large-3")
	if titleDiv == nil {
		titleDiv = findByClassExact(item, "columns small-7 large-11")
	}
	if titleDiv == nil {
		return ""
	}

	lis := findAllNodes(titleDiv, "li")
	if len(lis) >= 2 {
		text := extractText(lis[1])
		text = regexp.MustCompile(`\(\d+\)`).ReplaceAllString(text, "")
		return strings.TrimSpace(text)
	}
	return ""
}

func extractTimeText(item *html.Node, lang string) string {
	timeDiv := findByClassExact(item, "columns small-3 large-2")
	if timeDiv == nil {
		timeDiv = findByClassExact(item, "columns small-7 large-11")
	}
	if timeDiv == nil {
		return ""
	}

	lis := findAllNodes(timeDiv, "li")
	if len(lis) > 0 {
		return extractText(lis[0])
	}
	return ""
}

func extractDurationText(item *html.Node, lang string) string {
	timeDiv := findByClassExact(item, "columns small-3 large-2")
	if timeDiv == nil {
		timeDiv = findByClassExact(item, "columns small-7 large-11")
	}
	if timeDiv == nil {
		return ""
	}

	lis := findAllNodes(timeDiv, "li")
	if len(lis) >= 2 {
		spans := findAllNodes(lis[1], "span")
		if len(spans) > 0 {
			return extractText(spans[0])
		}
		return extractText(lis[1])
	}
	return ""
}

func extractImage(item *html.Node) string {
	imgDiv := findByClassExact(item, "columns small-3 large-1")
	if imgDiv == nil {
		imgDiv = findByClassExact(item, "columns small-5 large-1")
	}
	if imgDiv == nil {
		return ""
	}
	img := findNode(imgDiv, "img")
	if img == nil {
		return ""
	}
	return getAttr(img, "data-src")
}

func extractDescription(item *html.Node) string {
	descDiv := findByClassExact(item, "columns small-12 large-6")
	if descDiv == nil {
		return ""
	}
	lis := findAllNodes(descDiv, "li")
	if len(lis) >= 3 {
		text := extractText(lis[2])
		text = strings.ReplaceAll(text, "...اقرأ المزيد", "")
		text = strings.ReplaceAll(text, "...Read more", "")
		return strings.TrimSpace(text)
	}
	return ""
}

func parseTime12h(s string, date time.Time) time.Time {
	m := timeRe.FindStringSubmatch(s)
	if m == nil {
		return time.Time{}
	}

	hour, _ := strconv.Atoi(m[1])
	minute, _ := strconv.Atoi(m[2])
	ampm := m[3]

	if ampm == "PM" || ampm == "مساءً" {
		if hour != 12 {
			hour += 12
		}
	} else if hour == 12 {
		hour = 0
	}

	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, time.UTC)
}

func parseDuration(s string) int {
	m := durationRe.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	d, _ := strconv.Atoi(m[1])
	return d
}

func findByClassExact(n *html.Node, class string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && a.Val == class {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findByClassExact(c, class); found != nil {
			return found
		}
	}
	return nil
}

func findAllByClass(n *html.Node, class string) []*html.Node {
	var result []*html.Node
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && strings.Contains(a.Val, class) {
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

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
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
	return strings.TrimSpace(sb.String())
}
