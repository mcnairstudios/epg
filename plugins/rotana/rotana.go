package rotana

import (
	"fmt"
	"io"
	"net/http"
	"sort"
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

var riyadh *time.Location

func init() {
	riyadh, _ = time.LoadLocation("Asia/Riyadh")
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

func (p *provider) Name() string        { return "rotana.net" }
func (p *provider) Description() string { return "Rotana TV Arabic channels" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "rotana.net")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://rotana.net/%s/streams?channel=%s&tz=",
		channel.Lang, channel.SiteID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36 OPR/104.0.0.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from rotana", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	items := ParseSchedule(body, date)

	for i := range items {
		detail := p.fetchDetail(channel, items[i].ProgramID)
		if detail != nil {
			items[i].Description = detail.Description
			items[i].Category = detail.Category
			items[i].Image = detail.Image
		}
	}

	return toPrograms(items, channel), nil
}

type ScheduleItem struct {
	ProgramID   string
	Title       string
	Description string
	Category    string
	Image       string
	Start       time.Time
	Stop        time.Time
}

type ProgramDetail struct {
	Description string
	Category    string
	Image       string
}

func ParseSchedule(data []byte, date time.Time) []ScheduleItem {
	if len(data) == 0 {
		return nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil
	}

	hourDiv := findByClass(doc, "hour")
	if hourDiv == nil {
		return nil
	}

	var items []ScheduleItem
	var curDate string

	for c := hourDiv.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || c.Data != "div" {
			continue
		}

		cls := getAttr(c, "class")

		if strings.Contains(cls, "bg") {
			id := getAttr(c, "id")
			parts := strings.SplitN(id, "-", 2)
			if len(parts) == 2 {
				curDate = parts[1]
			}
			continue
		}

		if !strings.Contains(cls, "iq-accordion") {
			continue
		}

		block := findByClass(c, "iq-accordion-block")
		if block == nil {
			continue
		}

		progID := getAttr(block, "id")
		if strings.HasPrefix(progID, "item-") {
			progID = progID[5:]
		}

		titleNode := findByClass(block, "iq-accordion-title")
		if titleNode == nil {
			continue
		}

		h5 := findFirstElement(titleNode, "h5")
		if h5 == nil {
			continue
		}

		spans := findAllElements(h5, "span")
		var timeStr, title string
		for i, span := range spans {
			cls := getAttr(span, "class")
			if strings.Contains(cls, "meridiem") || strings.Contains(cls, "daylight") || strings.Contains(cls, "egp-live") {
				continue
			}
			text := normalizeWhitespace(extractText(span))
			if i == 0 || (timeStr == "" && isTimeString(text)) {
				timeStr = text
			} else if text != "" {
				title = text
			}
		}

		if timeStr == "" || curDate == "" {
			continue
		}

		dateParts := strings.Split(curDate, "-")
		if len(dateParts) != 3 {
			continue
		}

		startStr := fmt.Sprintf("%s-%s-%s %s", dateParts[2], dateParts[1], dateParts[0], timeStr)

		items = append(items, ScheduleItem{
			ProgramID: progID,
			Title:     title,
			Start:     parseRotanaTime(startStr),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Start.Before(items[j].Start)
	})

	for i := range items {
		if i < len(items)-1 {
			items[i].Stop = items[i+1].Start
		} else {
			next := items[i].Start.In(riyadh).Add(24 * time.Hour)
			midnight := time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, riyadh)
			items[i].Stop = midnight.UTC()
		}
	}

	dateStr := date.Format("2006-01-02")
	var filtered []ScheduleItem
	for _, item := range items {
		startDate := item.Start.In(riyadh).Format("2006-01-02")
		stopDate := item.Stop.In(riyadh).Format("2006-01-02")
		if startDate == dateStr || stopDate == dateStr {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

func ParseProgramDetail(data []byte) *ProgramDetail {
	if len(data) == 0 {
		return nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil
	}

	detail := &ProgramDetail{}

	trendingInfo := findByClass(doc, "trending-info")
	if trendingInfo != nil {
		rowDiv := findFirstElement(trendingInfo, "div")
		if rowDiv != nil {
			colDiv := findFirstElement(rowDiv, "div")
			if colDiv != nil {
				spans := findAllElements(colDiv, "span")
				for _, span := range spans {
					parent := span.Parent
					if parent == nil {
						continue
					}
					text := strings.TrimSpace(extractText(span))
					if strings.Contains(text, ":") {
						parts := strings.SplitN(text, ":", 2)
						key := strings.TrimSpace(parts[0])
						val := strings.TrimSpace(parts[1])
						switch key {
						case "Category", "التصنيف":
							detail.Category = val
						}
					}
				}

				outerSpan := findDirectSpanChild(colDiv)
				if outerSpan != nil {
					for c := outerSpan.FirstChild; c != nil; c = c.NextSibling {
						if c.Type == html.TextNode {
							text := strings.TrimSpace(c.Data)
							if text != "" && detail.Description == "" {
								detail.Description = text
							}
						}
					}
				}
			}
		}
	}

	rows := findAllByClass(doc, "row")
	for _, row := range rows {
		for c := row.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "div" {
				img := findFirstElement(c, "img")
				if img != nil {
					src := getAttr(img, "src")
					if src != "" && detail.Image == "" {
						detail.Image = src
					}
				}
			}
		}
	}

	return detail
}

func (p *provider) fetchDetail(channel domain.Channel, programID string) *ProgramDetail {
	url := fmt.Sprintf("https://rotana.net/%s/streams?channel=%s&itemId=%s",
		channel.Lang, channel.SiteID, programID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36 OPR/104.0.0.0")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	return ParseProgramDetail(body)
}

func toPrograms(items []ScheduleItem, channel domain.Channel) []domain.Program {
	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "rotana.net",
			Title:       item.Title,
			Description: item.Description,
			Category:    item.Category,
			Image:       item.Image,
			Lang:        channel.Lang,
			Start:       item.Start,
			Stop:        item.Stop,
		})
	}
	return programs
}

func parseRotanaTime(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", strings.TrimSpace(s), riyadh)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func normalizeWhitespace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

func isTimeString(s string) bool {
	for _, c := range s {
		if c != ':' && (c < '0' || c > '9') && c != ' ' {
			return false
		}
	}
	return strings.Contains(s, ":")
}

func findDirectSpanChild(n *html.Node) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "span" {
			return c
		}
	}
	return nil
}

func findAllByClass(n *html.Node, class string) []*html.Node {
	var result []*html.Node
	if n.Type == html.ElementNode {
		cls := getAttr(n, "class")
		for _, c := range strings.Fields(cls) {
			if c == class {
				result = append(result, n)
				break
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, findAllByClass(c, class)...)
	}
	return result
}

func findByClass(n *html.Node, class string) *html.Node {
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

func findFirstElement(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findFirstElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func findAllElements(n *html.Node, tag string) []*html.Node {
	var result []*html.Node
	if n.Type == html.ElementNode && n.Data == tag {
		result = append(result, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, findAllElements(c, tag)...)
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

func (p *provider) Timezone() string { return "Asia/Riyadh" }
