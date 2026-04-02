package telebilbao

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
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

var madrid *time.Location

var spanishDays = map[time.Weekday]string{
	time.Monday:    "LUNES",
	time.Tuesday:   "MARTES",
	time.Wednesday: "MIERCOLES",
	time.Thursday:  "JUEVES",
	time.Friday:    "VIERNES",
	time.Saturday:  "SABADO",
	time.Sunday:    "DOMINGO",
}

var timeRe = regexp.MustCompile(`^\d{1,2}:\d{2}$`)

func init() {
	madrid, _ = time.LoadLocation("Europe/Madrid")
	plugin.Register(&provider{client: httpclient.Browser()})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "telebilbao.es" }
func (p *provider) Description() string { return "TeleBilbao Spanish TV schedule" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "telebilbao.es")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	req, err := http.NewRequest(http.MethodGet, "https://www.telebilbao.es/programacion-2/", nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from telebilbao", resp.StatusCode)
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
		return nil, err
	}

	table := findByClass(doc, "programacion")
	if table == nil {
		return []domain.Program{}, nil
	}

	grid := flattenTable(table)
	if len(grid) < 2 {
		return []domain.Program{}, nil
	}

	targetDay := spanishDays[date.Weekday()]
	colIdx := -1
	for i, cell := range grid[0] {
		upper := strings.ToUpper(strings.TrimSpace(cell))
		if strings.Contains(upper, targetDay) {
			colIdx = i
			break
		}
	}

	if colIdx < 0 {
		return []domain.Program{}, nil
	}

	grid = grid[1:]
	if len(grid) > 0 {
		grid = grid[:len(grid)-1]
	}

	var items []item
	for _, row := range grid {
		if colIdx >= len(row) {
			continue
		}
		t := normalizeTime(strings.TrimSpace(row[0]))
		title := strings.TrimSpace(row[colIdx])
		if t == "" {
			continue
		}
		items = append(items, item{timeStr: t, title: title})
	}

	dateStr := date.Format("2006-01-02")
	currentDate := date
	var programs []domain.Program

	for _, it := range items {
		start, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+it.timeStr, madrid)
		if err != nil {
			continue
		}

		if len(programs) > 0 && start.Before(programs[len(programs)-1].Start) {
			currentDate = currentDate.AddDate(0, 0, 1)
			dateStr = currentDate.Format("2006-01-02")
			start, err = time.ParseInLocation("2006-01-02 15:04", dateStr+" "+it.timeStr, madrid)
			if err != nil {
				continue
			}
		}

		if len(programs) > 0 {
			programs[len(programs)-1].Stop = start.UTC()
		}

		stop := start.Add(30 * time.Minute)

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "telebilbao.es",
			Title:   it.title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		})
	}

	return programs, nil
}

type item struct {
	timeStr string
	title   string
}

func flattenTable(table *html.Node) [][]string {
	var trs []*html.Node
	collectElements(table, "tr", &trs)

	if len(trs) == 0 {
		return nil
	}

	maxCols := 0
	for _, tr := range trs {
		cols := 0
		for c := tr.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
				cols += getSpan(c, "colspan")
			}
		}
		if cols > maxCols {
			maxCols = cols
		}
	}
	if maxCols == 0 {
		return nil
	}

	grid := make([][]string, len(trs))
	for i := range grid {
		grid[i] = make([]string, maxCols)
	}

	occupied := make([][]bool, len(trs))
	for i := range occupied {
		occupied[i] = make([]bool, maxCols)
	}

	for r, tr := range trs {
		col := 0
		for c := tr.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.ElementNode || (c.Data != "td" && c.Data != "th") {
				continue
			}

			for col < maxCols && occupied[r][col] {
				col++
			}
			if col >= maxCols {
				break
			}

			text := strings.TrimSpace(extractText(c))
			rs := getSpan(c, "rowspan")
			cs := getSpan(c, "colspan")

			for dr := 0; dr < rs && r+dr < len(trs); dr++ {
				for dc := 0; dc < cs && col+dc < maxCols; dc++ {
					grid[r+dr][col+dc] = text
					occupied[r+dr][col+dc] = true
				}
			}

			col += cs
		}
	}

	return grid
}

func getSpan(n *html.Node, attr string) int {
	s := getAttr(n, attr)
	if s == "" {
		return 1
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return 1
	}
	return v
}

func collectElements(n *html.Node, tag string, result *[]*html.Node) {
	if n.Type == html.ElementNode && n.Data == tag {
		*result = append(*result, n)
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectElements(c, tag, result)
	}
}

func findByClass(n *html.Node, class string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" {
				for _, c := range strings.Fields(a.Val) {
					if c == class {
						return n
					}
				}
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

func normalizeTime(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "..", ":")
	s = strings.ReplaceAll(s, "::", ":")
	s = strings.ReplaceAll(s, ".", ":")
	s = strings.ReplaceAll(s, "_", ":")
	if !timeRe.MatchString(s) {
		return ""
	}
	return s
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func (p *provider) Timezone() string { return "Europe/Madrid" }
