package tvireland

import (
	"fmt"
	"io"
	"net/http"
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

func (p *provider) Name() string        { return "tvireland.ie" }
func (p *provider) Description() string { return "TV Ireland Irish television listings" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvireland.ie")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	u := fmt.Sprintf("https://www.tvireland.ie/tv/listings/channel/%s?dt=%s", channel.SiteID, date.Format("2006-01-02"))

	req, err := http.NewRequest(http.MethodGet, u, nil)
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

	return Parse(body, channel, date)
}

var dublin *time.Location

func init() {
	var err error
	dublin, err = time.LoadLocation("Europe/Dublin")
	if err != nil {
		dublin = time.UTC
	}
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	rows := parseRows(string(data))
	if len(rows) == 0 {
		return []domain.Program{}, nil
	}

	var programs []domain.Program
	for _, row := range rows {
		timeStr := strings.TrimSpace(row.time)
		if timeStr == "" {
			continue
		}

		dateStr := date.Format("2006-01-02") + " " + timeStr
		start := parseTimeStr(dateStr, dublin)
		if start.IsZero() {
			continue
		}

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				date = date.AddDate(0, 0, 1)
				dateStr = date.Format("2006-01-02") + " " + timeStr
				start = parseTimeStr(dateStr, dublin)
			}
			prev.Stop = start
		}

		stop := start.Add(30 * time.Minute)

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "tvireland.ie",
			Title:   row.title,
			Lang:    "en",
			Start:   start,
			Stop:    stop,
		})
	}

	return programs, nil
}

func parseTimeStr(dateStr string, loc *time.Location) time.Time {
	dateStr = strings.ToLower(strings.TrimSpace(dateStr))
	dateStr = strings.ReplaceAll(dateStr, "am", " AM")
	dateStr = strings.ReplaceAll(dateStr, "pm", " PM")
	dateStr = strings.TrimSpace(dateStr)

	t, err := time.ParseInLocation("2006-01-02 3:04 PM", dateStr, loc)
	if err != nil {
		return time.Time{}
	}
	return t
}

type rowData struct {
	time  string
	title string
}

func parseRows(content string) []rowData {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil
	}

	table := findTableWithClass(doc, "table")
	if table == nil {
		return nil
	}

	tbody := findChild(table, "tbody")
	if tbody == nil {
		return nil
	}

	var rows []rowData
	for c := tbody.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || c.Data != "tr" {
			continue
		}
		row := parseRow(c)
		if row.title != "" {
			rows = append(rows, row)
		}
	}
	return rows
}

func parseRow(tr *html.Node) rowData {
	var tds []*html.Node
	for c := tr.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "td" {
			tds = append(tds, c)
		}
	}

	var row rowData
	if len(tds) >= 1 {
		row.time = strings.TrimSpace(getText(tds[0]))
	}
	if len(tds) >= 2 {
		row.title = strings.TrimSpace(getText(tds[1]))
	}
	return row
}

func findTableWithClass(n *html.Node, class string) *html.Node {
	if n.Type == html.ElementNode && n.Data == "table" && hasClass(n, class) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findTableWithClass(c, class); result != nil {
			return result
		}
	}
	return nil
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

func findChild(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			return c
		}
	}
	return nil
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
