package mytelly

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

var londonTZ = func() *time.Location {
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		return time.UTC
	}
	return loc
}()

var seasonEpisodeRe = regexp.MustCompile(`Season (\d+), Episode (\d+)`)

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "mytelly.co.uk" }
func (p *provider) Description() string { return "My Telly UK TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "mytelly.co.uk")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	parts := strings.SplitN(channel.SiteID, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid site_id: %s", channel.SiteID)
	}

	u := fmt.Sprintf(
		"https://www.mytelly.co.uk/tv-guide/listings/channel/%s/%s.html?dt=%s",
		parts[0], parts[1], date.Format("2006-01-02"),
	)

	body, err := p.fetch(u)
	if err != nil {
		return nil, err
	}

	urls := ParseListing(body)
	if len(urls) == 0 {
		return []domain.Program{}, nil
	}

	var programs []domain.Program
	for _, progURL := range urls {
		progBody, err := p.fetch(progURL)
		if err != nil {
			continue
		}
		prog := ParseProgramme(progBody, channel, date)
		if prog.Title != "" {
			programs = append(programs, prog)
		}
	}

	return programs, nil
}

func (p *provider) fetch(u string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 OPR/117.0.0.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func ParseListing(data []byte) []string {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil
	}

	table := findNodeByTagAndClass(doc, "table", "table")
	if table == nil {
		return nil
	}

	tbody := findChild(table, "tbody")
	if tbody == nil {
		return nil
	}

	var urls []string
	for tr := tbody.FirstChild; tr != nil; tr = tr.NextSibling {
		if tr.Type != html.ElementNode || tr.Data != "tr" {
			continue
		}

		tds := collectChildren(tr, "td")
		if len(tds) < 2 {
			continue
		}

		h5 := findChild(tds[1], "h5")
		if h5 == nil {
			continue
		}

		a := findChild(h5, "a")
		if a == nil {
			continue
		}

		href := getAttr(a, "href")
		if href != "" {
			urls = append(urls, href)
		}
	}

	return urls
}

func ParseProgramme(data []byte, channel domain.Channel, date time.Time) domain.Program {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return domain.Program{}
	}

	prog := domain.Program{
		Channel: channel.XmltvID,
		Source:  "mytelly.co.uk",
		Lang:    channel.Lang,
	}

	prog.Title = extractTitle(doc)

	timeText := extractTimeText(doc)
	if timeText != "" {
		start, stop := parseStartStop(timeText, date)
		prog.Start = start
		prog.Stop = stop
	}

	prog.Image = extractImage(doc)
	prog.SubTitle = extractSubTitle(doc)
	prog.Description = extractDescription(doc)

	categories := extractCategories(doc)
	if len(categories) > 0 {
		prog.Category = strings.Join(categories, ",")
	}

	if prog.SubTitle != "" {
		matches := seasonEpisodeRe.FindStringSubmatch(prog.SubTitle)
		if len(matches) == 3 {
			prog.Season, _ = strconv.Atoi(matches[1])
			prog.Episode, _ = strconv.Atoi(matches[2])
		}
	}

	return prog
}

func extractTitle(doc *html.Node) string {
	node := findNodeByTagAndClass(doc, "div", "inner-heading")
	if node == nil {
		return ""
	}
	h2 := findDescendant(node, "h2")
	if h2 == nil {
		return ""
	}
	return cleanText(getText(h2))
}

func extractTimeText(doc *html.Node) string {
	var found *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "center" {
			h5 := findChild(n, "h5")
			if h5 != nil {
				b := findChild(h5, "b")
				if b != nil {
					found = b
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	if found == nil {
		return ""
	}
	return cleanText(getText(found))
}

func extractImage(doc *html.Node) string {
	node := findNodeByTagAndClass(doc, "div", "program-media-image")
	if node == nil {
		return ""
	}
	img := findDescendant(node, "img")
	if img == nil {
		return ""
	}
	return getAttr(img, "src")
}

func extractSubTitle(doc *html.Node) string {
	tabPanes := findAllNodesByTagAndClass(doc, "div", "tab-pane")
	for _, pane := range tabPanes {
		h5 := findChild(pane, "h5")
		if h5 == nil {
			continue
		}
		strong := findChild(h5, "strong")
		if strong == nil {
			continue
		}
		text := cleanText(getText(strong))
		if text != "" {
			return text
		}
	}
	return ""
}

func extractDescription(doc *html.Node) string {
	divs := findAllNodesByTagAndClass(doc, "div", "tvbody")
	for _, div := range divs {
		p := findChild(div, "p")
		if p != nil {
			text := cleanText(getText(p))
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func extractCategories(doc *html.Node) []string {
	node := findNodeByTagAndClass(doc, "div", "schedule-attributes-genres")
	if node == nil {
		return nil
	}

	var cats []string
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "span" {
			text := cleanText(getText(c))
			if text != "" {
				cats = append(cats, text)
			}
		}
	}
	return cats
}

func parseStartStop(timeStr string, date time.Time) (time.Time, time.Time) {
	parts := strings.Split(timeStr, " - ")
	if len(parts) != 2 {
		return time.Time{}, time.Time{}
	}

	start := parseTime(date, strings.TrimSpace(parts[0]))
	stop := parseTime(date, strings.TrimSpace(parts[1]))

	if stop.Before(start) {
		stop = stop.AddDate(0, 0, 1)
	}

	return start, stop
}

func parseTime(date time.Time, timeStr string) time.Time {
	dateStr := date.Format("2006-01-02") + " " + timeStr
	dateStr = strings.ToLower(strings.TrimSpace(dateStr))
	dateStr = strings.ReplaceAll(dateStr, "am", " AM")
	dateStr = strings.ReplaceAll(dateStr, "pm", " PM")
	dateStr = strings.TrimSpace(dateStr)

	t, err := time.ParseInLocation("2006-01-02 3:04 PM", dateStr, londonTZ)
	if err != nil {
		return time.Time{}
	}
	return t
}

func findNodeByTagAndClass(n *html.Node, tag, class string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag && hasClass(n, class) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findNodeByTagAndClass(c, tag, class); result != nil {
			return result
		}
	}
	return nil
}

func findAllNodesByTagAndClass(n *html.Node, tag, class string) []*html.Node {
	var results []*html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == tag && hasClass(node, class) {
			results = append(results, node)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return results
}

func findChild(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			return c
		}
	}
	return nil
}

func findDescendant(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findDescendant(c, tag); result != nil {
			return result
		}
	}
	return nil
}

func collectChildren(n *html.Node, tag string) []*html.Node {
	var nodes []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			nodes = append(nodes, c)
		}
	}
	return nodes
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

func getText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(getText(c))
	}
	return sb.String()
}

func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\t", "")
	s = strings.ReplaceAll(s, "\n", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}
