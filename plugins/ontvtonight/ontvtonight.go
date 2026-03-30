package ontvtonight

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
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

func (p *provider) Name() string        { return "ontvtonight.com" }
func (p *provider) Description() string { return "OnTVTonight television listings" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "ontvtonight.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	u := buildURL(channel, date)

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from ontvtonight", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

func buildURL(channel domain.Channel, date time.Time) string {
	parts := strings.SplitN(channel.SiteID, "#", 2)
	region := parts[0]
	id := parts[1]

	u := "https://www.ontvtonight.com"
	if region != "" && region != "us" {
		u += "/" + region
	}
	u += fmt.Sprintf("/guide/listings/channel/%s.html?dt=%s", id, date.Format("2006-01-02"))
	return u
}

var tzMap = map[string]string{
	"au": "Australia/Sydney",
	"ca": "America/Toronto",
	"us": "America/New_York",
}

var timeRe = regexp.MustCompile(`(\d{1,2}:\d{2}\s*[aApP][mM])`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	rows := parseRows(string(data))
	if len(rows) == 0 {
		return []domain.Program{}, nil
	}

	region := strings.SplitN(channel.SiteID, "#", 2)[0]
	tzName := tzMap[region]
	if tzName == "" {
		tzName = "UTC"
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		loc = time.UTC
	}

	var programs []domain.Program
	for _, row := range rows {
		timeStr := strings.TrimSpace(row.time)
		if timeStr == "" {
			continue
		}

		dateStr := date.Format("2006-01-02") + " " + timeStr
		start := parseTimeStr(dateStr, loc)
		if start.IsZero() {
			continue
		}

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				date = date.AddDate(0, 0, 1)
				dateStr = date.Format("2006-01-02") + " " + timeStr
				start = parseTimeStr(dateStr, loc)
			}
			prev.Stop = start
		}

		stop := start.Add(1 * time.Hour)

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "ontvtonight.com",
			Title:       row.title,
			Description: row.description,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
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
	time        string
	title       string
	description string
}

func parseRows(content string) []rowData {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil
	}

	table := findTable(doc)
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
		row.time = getH5Text(tds[0])
	}
	if len(tds) >= 2 {
		row.title = getH5Text(tds[1])
		row.description = getH6Text(tds[1])
	}
	return row
}

func getH5Text(td *html.Node) string {
	h5 := findChild(td, "h5")
	if h5 == nil {
		return ""
	}
	return strings.TrimSpace(getText(h5))
}

func getH6Text(td *html.Node) string {
	h6 := findChild(td, "h6")
	if h6 == nil {
		return ""
	}
	return strings.TrimSpace(getText(h6))
}

func findTable(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "table" {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findTable(c); result != nil {
			return result
		}
	}
	return nil
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
