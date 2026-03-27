package clickthecity

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
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

//go:embed channels.json
var channelsData []byte

var manila *time.Location

func init() {
	manila, _ = time.LoadLocation("Asia/Manila")
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

func (p *provider) Name() string        { return "clickthecity.com" }
func (p *provider) Description() string { return "ClickTheCity Philippines TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "clickthecity.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	apiURL := fmt.Sprintf("https://www.clickthecity.com/tv/channels/?netid=%s", channel.SiteID)

	manilaDate := date.In(manila).Format("2006-01-02")
	form := url.Values{}
	form.Set("optDate", manilaDate)
	form.Set("optTime", "00:00:00")

	req, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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

var startTimeRegex = regexp.MustCompile(`starttime=(\d{1,2})%3A(\d{2})\+(AM|PM)`)
var endTimeRegex = regexp.MustCompile(`endtime=(\d{1,2})%3A(\d{2})\+(AM|PM)`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	table := findByID(doc, "tvlistings")
	if table == nil {
		return []domain.Program{}, nil
	}

	tbody := findNode(table, "tbody")
	if tbody == nil {
		return []domain.Program{}, nil
	}

	rows := findAllDirectChildren(tbody, "tr")
	var programs []domain.Program

	for _, row := range rows {
		if !hasCPrgCell(row) {
			continue
		}

		title := extractTitle(row)
		if title == "" {
			continue
		}

		start := extractStartTime(row, date)
		stop := extractStopTime(row, date)
		if start.IsZero() || stop.IsZero() {
			continue
		}

		if start.After(stop) {
			stop = stop.AddDate(0, 0, 1)
		}

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "clickthecity.com",
			Title:   title,
			Lang:    channel.Lang,
			Start:   start,
			Stop:    stop,
		})
	}

	return programs, nil
}

func extractTitle(row *html.Node) string {
	for c := row.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "td" {
			a := findNode(c, "a")
			if a != nil {
				text := strings.TrimSpace(extractText(a))
				if text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func extractStartTime(row *html.Node, date time.Time) time.Time {
	href := findCPrgHref(row)
	if href == "" {
		return time.Time{}
	}
	m := startTimeRegex.FindStringSubmatch(href)
	if m == nil {
		return time.Time{}
	}
	return parseAMPMTime(m[1], m[2], m[3], date)
}

func extractStopTime(row *html.Node, date time.Time) time.Time {
	href := findCPrgHref(row)
	if href == "" {
		return time.Time{}
	}
	m := endTimeRegex.FindStringSubmatch(href)
	if m == nil {
		return time.Time{}
	}
	return parseAMPMTime(m[1], m[2], m[3], date)
}

func parseAMPMTime(hourStr, minStr, ampm string, date time.Time) time.Time {
	hour, minute := 0, 0
	fmt.Sscanf(hourStr, "%d", &hour)
	fmt.Sscanf(minStr, "%d", &minute)

	if ampm == "PM" && hour != 12 {
		hour += 12
	} else if ampm == "AM" && hour == 12 {
		hour = 0
	}

	t := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, manila)
	return t.UTC()
}

func findCPrgHref(row *html.Node) string {
	cprg := findCPrgCell(row)
	if cprg == nil {
		return ""
	}
	a := findNode(cprg, "a")
	if a == nil {
		return ""
	}
	return getAttr(a, "href")
}

func hasCPrgCell(row *html.Node) bool {
	return findCPrgCell(row) != nil
}

func findCPrgCell(row *html.Node) *html.Node {
	for c := row.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "td" {
			cls := getAttr(c, "class")
			if strings.Contains(cls, "cPrg") {
				return c
			}
		}
	}
	return nil
}

func findByID(n *html.Node, id string) *html.Node {
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

func findAllDirectChildren(n *html.Node, tag string) []*html.Node {
	var result []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			result = append(result, c)
		}
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
