package getafteritmedia

import (
	"io"
	"net/http"
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

var eastern *time.Location

func init() {
	var err error
	eastern, err = time.LoadLocation("America/New_York")
	if err != nil {
		eastern = time.FixedZone("EST", -5*60*60)
	}
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

func (p *provider) Name() string { return "getafteritmedia.com" }
func (p *provider) Description() string {
	return "Get After It Media TV listings"
}

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "getafteritmedia.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := "https://docs.google.com/spreadsheets/d/e/2PACX-1vQcDmb9OnO0HpbjINfGaepqgGTp3VSmPs7hs654n3sRKrq4Q9y6uPSEvVvq9MwTLYG_n_V7vh0rFYP9/pubhtml"

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

	sheetDiv := findByID(doc, channel.SiteID)
	if sheetDiv == nil {
		return []domain.Program{}, nil
	}

	table := findNode(sheetDiv, "table")
	if table == nil {
		return []domain.Program{}, nil
	}

	rows := extractTableRows(table)
	if len(rows) <= 5 {
		return []domain.Program{}, nil
	}
	rows = rows[5:]

	dayCol := isoWeekday(date) + 1

	var programs []domain.Program
	for _, row := range rows {
		if len(row) <= dayCol || len(row) < 2 {
			continue
		}

		timeStr := strings.TrimSpace(row[1])
		title := strings.TrimSpace(row[dayCol])
		if timeStr == "" || title == "" {
			continue
		}

		start := parseAmPmTime(timeStr, date)
		if start.IsZero() {
			continue
		}

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				date = date.AddDate(0, 0, 1)
				start = parseAmPmTime(timeStr, date)
			}
			prev.Stop = start
		}

		stop := start.Add(30 * time.Minute)
		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "getafteritmedia.com",
			Title:   title,
			Lang:    channel.Lang,
			Start:   start,
			Stop:    stop,
		})
	}

	if programs == nil {
		return []domain.Program{}, nil
	}
	return programs, nil
}

func parseAmPmTime(timeStr string, date time.Time) time.Time {
	timeStr = strings.TrimSpace(timeStr)

	parts := strings.Fields(timeStr)
	if len(parts) != 2 {
		return time.Time{}
	}

	timePart := parts[0]
	ampm := strings.ToUpper(parts[1])

	hm := strings.Split(timePart, ":")
	if len(hm) != 2 {
		return time.Time{}
	}

	hour, err := strconv.Atoi(hm[0])
	if err != nil {
		return time.Time{}
	}
	minute, err := strconv.Atoi(hm[1])
	if err != nil {
		return time.Time{}
	}

	if ampm == "AM" && hour == 12 {
		hour = 0
	} else if ampm == "PM" && hour != 12 {
		hour += 12
	}

	t := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, eastern)
	return t.UTC()
}

func isoWeekday(t time.Time) int {
	d := int(t.Weekday())
	if d == 0 {
		return 7
	}
	return d
}

func findByID(n *html.Node, id string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == id {
				return n
			}
		}
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

func extractTableRows(table *html.Node) [][]string {
	var rows [][]string
	var trs []*html.Node
	findTRs(table, &trs)

	for _, tr := range trs {
		var cells []string
		findCells(tr, &cells)
		rows = append(rows, cells)
	}
	return rows
}

func findTRs(n *html.Node, trs *[]*html.Node) {
	if n.Type == html.ElementNode && n.Data == "tr" {
		*trs = append(*trs, n)
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		findTRs(c, trs)
	}
}

func findCells(n *html.Node, cells *[]string) {
	if n.Type == html.ElementNode && (n.Data == "td" || n.Data == "th") {
		*cells = append(*cells, strings.TrimSpace(extractText(n)))
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		findCells(c, cells)
	}
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
