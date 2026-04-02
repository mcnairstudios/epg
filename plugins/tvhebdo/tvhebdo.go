package tvhebdo

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

var toronto *time.Location

func init() {
	var err error
	toronto, err = time.LoadLocation("America/Toronto")
	if err != nil {
		panic(err)
	}

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

func (p *provider) Name() string        { return "tvhebdo.com" }
func (p *provider) Description() string { return "TV Hebdo Canadian French TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvhebdo.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Format("2006-01-02")
	url := fmt.Sprintf("https://www.tvhebdo.com/horaire-tele/%s/date/%s", channel.SiteID, d)

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
		return nil, fmt.Errorf("HTTP %d from tvhebdo", resp.StatusCode)
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

	rows := findProgramRows(doc)
	if len(rows) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(rows))

	for _, row := range rows {
		timeText := extractClassText(row, "heure")
		titleText := extractClassText(row, "titre")

		if timeText == "" || titleText == "" {
			continue
		}

		start := parseTime(date, timeText)

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				start = start.Add(24 * time.Hour)
			}
			prev.Stop = start
		}

		stop := start.Add(30 * time.Minute)

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "tvhebdo.com",
			Title:   titleText,
			Lang:    channel.Lang,
			Start:   start,
			Stop:    stop,
		})
	}

	return programs, nil
}

func parseTime(date time.Time, timeStr string) time.Time {
	dateStr := date.Format("2006-01-02")
	t, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+timeStr, toronto)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func findProgramRows(doc *html.Node) []*html.Node {
	container := findByID(doc, "main_container")
	if container == nil {
		return nil
	}

	listDiv := findByClass(container, "liste_container")
	if listDiv == nil {
		return nil
	}

	table := findNode(listDiv, "table")
	if table == nil {
		return nil
	}

	tbody := findNode(table, "tbody")
	if tbody == nil {
		tbody = table
	}

	var rows []*html.Node
	for _, tr := range findAllNodes(tbody, "tr") {
		cls := getAttr(tr, "class")
		if strings.HasPrefix(cls, "liste_row_style_") {
			rows = append(rows, tr)
		}
	}
	return rows
}

func extractClassText(n *html.Node, class string) string {
	node := findByClass(n, class)
	if node == nil {
		return ""
	}
	return strings.TrimSpace(extractText(node))
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

func findByClass(n *html.Node, class string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && containsClass(a.Val, class) {
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

func containsClass(classList, target string) bool {
	for _, c := range strings.Fields(classList) {
		if c == target {
			return true
		}
	}
	return false
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

func (p *provider) Timezone() string { return "America/Toronto" }
