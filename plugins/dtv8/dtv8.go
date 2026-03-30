package dtv8

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
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

//go:embed channels.json
var channelsData []byte

var guyana *time.Location

func init() {
	guyana = time.FixedZone("America/Guyana", -4*60*60)
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

func (p *provider) Name() string        { return "dtv8.net" }
func (p *provider) Description() string { return "DTV-8 Guyana TV listings" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "dtv8.net")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	day := strings.ToLower(date.Format("Monday"))
	url := fmt.Sprintf("https://dtv8.net/tv-listings/%s/", day)

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
		return nil, fmt.Errorf("HTTP %d from dtv8", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

var timeRe = regexp.MustCompile(`(\d{1,2}:\d{2})\s*hrs\.`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	rows := findAllTR(doc)
	var programs []domain.Program

	for _, row := range rows {
		tds := findAllNodes(row, "td")
		if len(tds) < 2 {
			continue
		}

		tdText := extractText(tds[0])
		if tdText == "" || strings.Contains(tdText, "Time") {
			continue
		}

		timeStr := extractTimeFromText(tdText)
		if timeStr == "" {
			continue
		}

		title := extractTitle(tds[1])
		if title == "" {
			continue
		}

		start := parseTimeStr(timeStr, date)
		if start.IsZero() {
			continue
		}

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				date = date.AddDate(0, 0, 1)
				start = parseTimeStr(timeStr, date)
			}
			prev.Stop = start
		}

		stop := start.Add(30 * time.Minute)

		desc := extractDescription(tds[1])
		image := extractImage(tds[0])

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "dtv8.net",
			Title:       title,
			Description: desc,
			Image:       image,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

func parseTimeStr(timeStr string, date time.Time) time.Time {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return time.Time{}
	}
	hour, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	minute, _ := strconv.Atoi(strings.TrimSpace(parts[1]))

	t := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, guyana)
	return t.UTC()
}

func extractTimeFromText(text string) string {
	m := timeRe.FindStringSubmatch(text)
	if m != nil {
		return m[1]
	}
	return ""
}

func extractTitle(td *html.Node) string {
	for _, tag := range []string{"strong", "b"} {
		nodes := findAllNodes(td, tag)
		for _, n := range nodes {
			text := strings.TrimSpace(extractText(n))
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func extractDescription(td *html.Node) string {
	strongs := findAllNodes(td, "strong")
	bs := findAllNodes(td, "b")
	all := append(strongs, bs...)

	if len(all) < 2 {
		return ""
	}

	for i := 1; i < len(all); i++ {
		text := strings.TrimSpace(extractText(all[i]))
		if text != "" && text != strings.TrimSpace(extractText(all[0])) {
			return text
		}
	}
	return ""
}

func extractImage(td *html.Node) string {
	imgs := findAllNodes(td, "img")
	for _, img := range imgs {
		cls := getAttr(img, "class")
		if strings.Contains(cls, "size-full") {
			return getAttr(img, "src")
		}
	}
	return ""
}

func findAllTR(n *html.Node) []*html.Node {
	var result []*html.Node
	if n.Type == html.ElementNode && n.Data == "tr" {
		result = append(result, n)
		return result
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, findAllTR(c)...)
	}
	return result
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
