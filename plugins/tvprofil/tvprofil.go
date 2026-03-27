package tvprofil

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func (p *provider) Name() string        { return "tvprofil.com" }
func (p *provider) Description() string { return "TV Profil multi-country guide (9091 channels)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvprofil.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	parts := strings.SplitN(channel.SiteID, "#", 2)
	if len(parts) != 2 {
		return []domain.Program{}, nil
	}

	query := buildQuery(parts[1], date)
	fetchURL := fmt.Sprintf("https://tvprofil.com/%s/program/?%s", parts[0], query)

	req, err := http.NewRequest(http.MethodGet, fetchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://tvprofil.com/tvprogram/")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	content := string(data)

	htmlContent := extractProgramHTML(content)
	if htmlContent == "" {
		return []domain.Program{}, nil
	}

	return parseHTML(htmlContent, channel)
}

var jsonpRE = regexp.MustCompile(`^[^(]+\(([\s\S]*)\)$`)
var programRE = regexp.MustCompile(`"program"\s*:\s*"((?:[^"\\]|\\.)*)"\s*[,}]`)

func extractProgramHTML(content string) string {
	matches := jsonpRE.FindStringSubmatch(content)
	if len(matches) < 2 {
		return ""
	}

	jsonStr := matches[1]
	progMatches := programRE.FindStringSubmatch(jsonStr)
	if len(progMatches) < 2 {
		return ""
	}

	raw := progMatches[1]
	raw = strings.ReplaceAll(raw, `\/`, `/`)
	raw = strings.ReplaceAll(raw, `\"`, `"`)
	raw = strings.ReplaceAll(raw, `\\`, `\`)

	return raw
}

func parseHTML(htmlContent string, channel domain.Channel) ([]domain.Program, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return []domain.Program{}, nil
	}

	var programs []domain.Program
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "row") {
			ts := getAttr(n, "data-ts")
			dur := getAttr(n, "data-len")
			image := getAttr(n, "data-image")

			if ts != "" && dur != "" {
				timestamp, _ := strconv.ParseInt(ts, 10, 64)
				duration, _ := strconv.ParseInt(dur, 10, 64)

				start := time.Unix(timestamp, 0).UTC()
				stop := start.Add(time.Duration(duration) * time.Second)

				title := extractTitle(n)
				category := extractCategory(n)

				p := domain.Program{
					Channel:  channel.XmltvID,
					Source:   "tvprofil.com",
					Title:    title,
					Category: category,
					Image:    image,
					Lang:     channel.Lang,
					Start:    start,
					Stop:     stop,
				}

				programs = append(programs, p)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	if programs == nil {
		return []domain.Program{}, nil
	}

	return programs, nil
}

func extractTitle(n *html.Node) string {
	col := findSecondCol(n)
	if col == nil {
		return ""
	}

	var title string
	a := findNodeByTag(col, "a")
	if a != nil {
		title = extractText(a)
	}

	remaining := extractDirectText(col)
	title += remaining

	title = strings.ReplaceAll(title, "\u00ae", "")
	title = strings.TrimSpace(title)
	title = strings.TrimRight(title, ",")

	return title
}

func extractCategory(n *html.Node) string {
	col := findSecondCol(n)
	if col == nil {
		return ""
	}
	small := findNodeByTag(col, "small")
	if small == nil {
		return ""
	}
	return strings.TrimSpace(extractText(small))
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

func findSecondCol(n *html.Node) *html.Node {
	count := 0
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "div" && hasClass(c, "col") {
			count++
			if count == 2 {
				return c
			}
		}
	}
	return nil
}

func hasClass(n *html.Node, class string) bool {
	cls := getAttr(n, "class")
	return strings.Contains(cls, class)
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
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

func buildQuery(siteID string, date time.Time) string {
	datum := date.Format("2006-01-02")

	params := url.Values{}
	params.Set("datum", datum)
	params.Set("kanal", siteID)

	c := 4
	a := datum + siteID + strconv.Itoa(c)
	ua := siteID + datum

	if ua == "" || ua == "0" {
		ua = "none"
	}

	for j := 0; j < len(ua); j++ {
		c += int(ua[j])
	}

	i := len(a) - 1
	b := 2
	for i >= 0 {
		b += (int(a[i]) + c*2) * i
		i--
	}

	bStr := strconv.Itoa(b)
	lastCharCode := int(bStr[len(bStr)-1])
	key := "b" + strconv.Itoa(lastCharCode)
	params.Set("callback", fmt.Sprintf("tvprogramit%d", lastCharCode))
	params.Set(key, bStr)

	return params.Encode()
}
