package enaskylife

import (
	"fmt"
	"io"
	"net/http"
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

var seoulTZ = time.FixedZone("Asia/Seoul", 9*60*60)

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

func (p *provider) Name() string        { return "ena.skylifetv.co.kr" }
func (p *provider) Description() string { return "ENA SkyLife TV Korea" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "ena.skylifetv.co.kr")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("http://ena.skylifetv.co.kr/%s/?day=%s&sc_dvsn=U",
		channel.SiteID, date.Format("20060102"))

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
		return nil, fmt.Errorf("HTTP %d from enaskylife", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	table := findByClass(doc, "tbl_schedule")
	if table == nil {
		return []domain.Program{}, nil
	}

	tbody := findNodeByTag(table, "tbody")
	if tbody == nil {
		return []domain.Program{}, nil
	}

	rows := findAllTR(tbody)
	if len(rows) == 0 {
		return []domain.Program{}, nil
	}

	dateStr := date.Format("2006-01-02")
	var programs []domain.Program

	for _, row := range rows {
		timeText := extractCellText(row, "col1")
		title := extractTitleFromRow(row)
		rating := extractCellText(row, "col4")
		durationStr := extractCellText(row, "col5")

		if timeText == "" || title == "" {
			continue
		}

		start, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+timeText, seoulTZ)
		if err != nil {
			continue
		}

		duration := 30
		if d, err := strconv.Atoi(strings.TrimSpace(durationStr)); err == nil && d > 0 {
			duration = d
		}

		stop := start.Add(time.Duration(duration) * time.Minute)

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "ena.skylifetv.co.kr",
			Title:   title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		}

		if rating != "" {
			p.Category = "KMRB:" + rating
		}

		programs = append(programs, p)
	}

	if programs == nil {
		return []domain.Program{}, nil
	}

	return programs, nil
}

func findAllTR(n *html.Node) []*html.Node {
	var result []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "tr" {
			result = append(result, c)
		}
	}
	return result
}

func extractCellText(row *html.Node, class string) string {
	cell := findByClass(row, class)
	if cell == nil {
		return ""
	}
	return strings.TrimSpace(extractText(cell))
}

func extractTitleFromRow(row *html.Node) string {
	col2 := findByClass(row, "col2")
	if col2 == nil {
		return ""
	}
	tit := findByClass(col2, "tit")
	if tit == nil {
		return ""
	}
	return strings.TrimSpace(extractText(tit))
}

func findByClass(n *html.Node, class string) *html.Node {
	if n == nil {
		return nil
	}
	if n.Type == html.ElementNode {
		cls := getAttr(n, "class")
		for _, c := range strings.Fields(cls) {
			if c == class {
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

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func (p *provider) Timezone() string { return "Asia/Seoul" }
