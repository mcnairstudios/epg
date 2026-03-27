package znbc

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
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "znbc.co.zm" }
func (p *provider) Description() string { return "ZNBC Zambia TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "znbc.co.zm")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://www.znbc.co.zm/%s/", channel.SiteID)

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
		return nil, fmt.Errorf("HTTP %d for channel %s", resp.StatusCode, channel.SiteID)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel, date)
}

var timeProgRe = regexp.MustCompile(`^(\d{2}:\d{2})\s+(.+)`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation("Africa/Lusaka")
	if err != nil {
		return nil, err
	}

	dayOfWeek := strings.ToUpper(date.Format("Monday"))

	tableHTML := findDayTable(doc, dayOfWeek)
	if tableHTML == nil {
		return []domain.Program{}, nil
	}

	rows := findRows(tableHTML)
	items := extractItems(rows)

	programs := make([]domain.Program, 0, len(items))
	currentDate := date

	for _, item := range items {
		timeStr := fmt.Sprintf("%s %s", currentDate.Format("2006-01-02"), item.time)
		start, err := time.ParseInLocation("2006-01-02 15:04", timeStr, loc)
		if err != nil {
			continue
		}

		if len(programs) > 0 && start.Before(programs[len(programs)-1].Start) {
			start = start.Add(24 * time.Hour)
			currentDate = currentDate.Add(24 * time.Hour)
		}

		if len(programs) > 0 {
			programs[len(programs)-1].Stop = start.UTC()
		}

		stop := start.Add(30 * time.Minute)

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "znbc.co.zm",
			Title:   item.title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		}
		programs = append(programs, p)
	}

	return programs, nil
}

type programItem struct {
	time  string
	title string
}

func findDayTable(doc *html.Node, dayOfWeek string) *html.Node {
	var mobileTitle *html.Node
	var found bool

	var f func(*html.Node)
	f = func(n *html.Node) {
		if found {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "elementor-tab-mobile-title") {
			text := strings.TrimSpace(textContent(n))
			if strings.EqualFold(text, dayOfWeek) {
				mobileTitle = n
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	if mobileTitle == nil {
		return nil
	}

	for sib := mobileTitle.NextSibling; sib != nil; sib = sib.NextSibling {
		if sib.Type == html.ElementNode && sib.Data == "div" {
			found = true
			return sib
		}
	}

	return nil
}

func findRows(tableContainer *html.Node) []*html.Node {
	var rows []*html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			rows = append(rows, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(tableContainer)
	return rows
}

func extractItems(rows []*html.Node) []programItem {
	var items []programItem
	for _, row := range rows {
		text := strings.TrimSpace(normalizeSpaces(textContent(row)))
		matches := timeProgRe.FindStringSubmatch(text)
		if matches == nil {
			continue
		}
		title := strings.TrimSpace(matches[2])
		if title == "" {
			continue
		}
		items = append(items, programItem{time: matches[1], title: title})
	}
	return items
}

func normalizeSpaces(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ")
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

func hasClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			for _, c := range strings.Fields(attr.Val) {
				if c == class {
					return true
				}
			}
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
