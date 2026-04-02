package reportv

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
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

//go:embed channels.json
var channelsData []byte

var caracas *time.Location

func init() {
	caracas = time.FixedZone("VET", -4*60*60)
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

func (p *provider) Name() string        { return "reportv.com.ar" }
func (p *provider) Description() string { return "ReporTV Argentina (163 channels)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "reportv.com.ar")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	params := url.Values{}
	params.Set("idSenial", channel.SiteID)
	params.Set("Alineacion", "2694")
	params.Set("DiaDesde", date.Format("2006/01/02"))
	params.Set("HoraDesde", "00:00:00")

	req, err := http.NewRequest(http.MethodPost, "https://www.reportv.com.ar/buscador/ProgXSenial.php", strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from reportv", resp.StatusCode)
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

	items := parseItems(data, date)
	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	var programs []domain.Program
	for _, item := range items {
		p := domain.Program{
			Channel:  channel.XmltvID,
			Source:   "reportv.com.ar",
			Title:    item.title,
			Category: item.category,
			Lang:     channel.Lang,
			Start:    item.start,
			Stop:     item.stop,
		}
		programs = append(programs, p)
	}

	return programs, nil
}

type parsedItem struct {
	title    string
	category string
	start    time.Time
	stop     time.Time
}

var onclickRE = regexp.MustCompile(`detallePrograma\((\d+),(\d+),(\d+),(\d+),'([^']+)'\)`)

func parseItems(data []byte, date time.Time) []parsedItem {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil
	}

	dateStr := fmt.Sprintf("%02d %s %d",
		date.Day(),
		spanishMonth(date.Month()),
		date.Year())

	var items []parsedItem
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && hasClass(n, "trProg") {
			title := getAttr(n, "title")
			if strings.Contains(title, dateStr) {
				item := extractItem(n, date)
				if item != nil {
					items = append(items, *item)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return items
}

func extractItem(n *html.Node, date time.Time) *parsedItem {
	divs := findAllByTag(n, "div")
	if len(divs) < 4 {
		return nil
	}

	firstSpan := findNodeByTag(divs[0], "span")
	if firstSpan == nil {
		return nil
	}
	text := strings.TrimSpace(extractText(firstSpan))
	parts := strings.SplitN(text, " - ", 2)
	if len(parts) != 2 {
		return nil
	}

	timeStr := strings.TrimSpace(parts[0])
	title := strings.TrimSpace(parts[1])

	thirdSpan := findNodeByTag(divs[2], "span")
	category := ""
	if thirdSpan != nil {
		category = strings.TrimSpace(extractText(thirdSpan))
	}

	fourthSpan := findNodeByTag(divs[3], "span")
	duration := time.Duration(0)
	if fourthSpan != nil {
		durStr := strings.TrimSpace(extractText(fourthSpan))
		duration = parseDuration(durStr)
	}

	start, err := time.ParseInLocation("2006-01-02 15:04", fmt.Sprintf("%s %s", date.Format("2006-01-02"), timeStr), caracas)
	if err != nil {
		return nil
	}

	stop := start.Add(duration)

	return &parsedItem{
		title:    title,
		category: category,
		start:    start.UTC(),
		stop:     stop.UTC(),
	}
}

func parseDuration(s string) time.Duration {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0
	}
	h, m, sec := 0, 0, 0
	fmt.Sscanf(parts[0], "%d", &h)
	fmt.Sscanf(parts[1], "%d", &m)
	fmt.Sscanf(parts[2], "%d", &sec)
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second
}

func spanishMonth(m time.Month) string {
	months := map[time.Month]string{
		time.January:   "Enero",
		time.February:  "Febrero",
		time.March:     "Marzo",
		time.April:     "Abril",
		time.May:       "Mayo",
		time.June:      "Junio",
		time.July:      "Julio",
		time.August:    "Agosto",
		time.September: "Septiembre",
		time.October:   "Octubre",
		time.November:  "Noviembre",
		time.December:  "Diciembre",
	}
	return months[m]
}

func hasClass(n *html.Node, class string) bool {
	return strings.Contains(getAttr(n, "class"), class)
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

func findAllByTag(n *html.Node, tag string) []*html.Node {
	var result []*html.Node
	if n == nil {
		return result
	}
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

func (p *provider) Timezone() string { return "America/New_York" }
