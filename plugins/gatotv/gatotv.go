package gatotv

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

var est *time.Location

func init() {
	est = time.FixedZone("EST", -5*60*60)
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

func (p *provider) Name() string        { return "gatotv.com" }
func (p *provider) Description() string { return "GatoTV Spanish TV guide (475 channels)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "gatotv.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Format("2006-01-02")
	url := fmt.Sprintf("https://www.gatotv.com/canal/%s/%s", channel.SiteID, d)

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
		return nil, fmt.Errorf("HTTP %d from gatotv", resp.StatusCode)
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

	rows := findEPGRows(doc)
	if len(rows) == 0 {
		return []domain.Program{}, nil
	}

	prevDay := date.AddDate(0, 0, -1)
	currentDate := prevDay
	var programs []domain.Program

	for i, row := range rows {
		startTime := extractTimeAttr(row, 0)
		stopTime := extractTimeAttr(row, 1)
		title := extractTitle(row)

		if startTime == "" || stopTime == "" || title == "" {
			continue
		}

		start := parseTime(startTime, currentDate)

		if i == 0 && start.Hour() >= 5 {
			currentDate = currentDate.AddDate(0, 0, 1)
			start = parseTime(startTime, currentDate)
		}

		stop := parseTime(stopTime, currentDate)
		if stop.Before(start) {
			currentDate = currentDate.AddDate(0, 0, 1)
			stop = parseTime(stopTime, currentDate)
		}

		description := extractDescription(row)

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "gatotv.com",
			Title:       title,
			Description: description,
			Image:       extractImage(row),
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

func parseTime(timeStr string, date time.Time) time.Time {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return time.Time{}
	}

	hour, minute := 0, 0
	fmt.Sscanf(parts[0], "%d", &hour)
	fmt.Sscanf(parts[1], "%d", &minute)

	t := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, est)
	return t.UTC()
}

func findEPGRows(n *html.Node) []*html.Node {
	table := findFirstEPGTable(n)
	if table == nil {
		return nil
	}

	var rows []*html.Node
	collectEPGRows(table, &rows)
	return rows
}

func findFirstEPGTable(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "table" {
		for _, a := range n.Attr {
			if a.Key == "class" && strings.TrimSpace(a.Val) == "tbl_EPG" {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findFirstEPGTable(c); found != nil {
			return found
		}
	}
	return nil
}

func collectEPGRows(n *html.Node, rows *[]*html.Node) {
	if n.Type == html.ElementNode && n.Data == "tr" {
		cls := getAttr(n, "class")
		trimmed := strings.TrimSpace(cls)
		if trimmed == "tbl_EPG_row" || trimmed == "tbl_EPG_rowAlternate" || trimmed == "tbl_EPG_row_selected" {
			if hasTimeElement(n) {
				*rows = append(*rows, n)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectEPGRows(c, rows)
	}
}

func hasTimeElement(n *html.Node) bool {
	if n.Type == html.ElementNode && n.Data == "time" {
		return true
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if hasTimeElement(c) {
			return true
		}
	}
	return false
}

func extractTimeAttr(row *html.Node, index int) string {
	count := 0
	return findTimeAttr(row, index, &count)
}

func findTimeAttr(n *html.Node, index int, count *int) string {
	if n.Type == html.ElementNode && n.Data == "time" {
		if *count == index {
			return getAttr(n, "datetime")
		}
		*count++
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findTimeAttr(c, index, count); result != "" {
			return result
		}
	}
	return ""
}

func extractTitle(row *html.Node) string {
	div := findByClass(row, "div_program_title_on_channel")
	if div == nil {
		return ""
	}
	span := findNode(div, "span")
	if span == nil {
		return ""
	}
	return strings.TrimSpace(extractText(span))
}

func extractDescription(row *html.Node) string {
	tds := findAllNodes(row, "td")
	if len(tds) < 3 {
		return ""
	}

	programTD := tds[2]
	programDiv := findByClass(programTD, "tbl_EPG_ProgramsColumn")
	if programDiv == nil {
		return ""
	}

	var parts []string
	for c := programDiv.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "div" {
			cls := getAttr(c, "class")
			if strings.Contains(cls, "div_program_title_on_channel") {
				continue
			}
			text := strings.TrimSpace(extractText(c))
			if text != "" {
				parts = append(parts, text)
			}
		}
	}

	return strings.Join(parts, " ")
}

func extractImage(row *html.Node) string {
	tds := findAllNodes(row, "td")
	if len(tds) < 3 {
		return ""
	}
	img := findNode(tds[2], "img")
	if img == nil {
		return ""
	}
	return getAttr(img, "src")
}

func findByClass(n *html.Node, class string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && strings.Contains(a.Val, class) {
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
