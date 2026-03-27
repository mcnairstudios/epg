package mncvision

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "mncvision.id" }
func (p *provider) Description() string { return "MNC Vision Indonesia TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "mncvision-154", Name: "AXN", Source: "mncvision.id", SiteID: "154", Lang: "id", XmltvID: "AXN.id"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	req, err := http.NewRequest(http.MethodPost, "https://www.mncvision.id/schedule/table", nil)
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
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, nil
	}

	rows := findRows(doc)
	if len(rows) == 0 {
		return nil, nil
	}

	loc, _ := time.LoadLocation("Asia/Jakarta")
	programs := make([]domain.Program, 0, len(rows))
	for _, row := range rows {
		cells := findCells(row)
		if len(cells) < 3 {
			continue
		}

		timeText := strings.TrimSpace(textContent(cells[0]))
		title := extractLinkText(cells[1])
		durationText := strings.TrimSpace(textContent(cells[2]))

		if title == "" || timeText == "" {
			continue
		}

		dateStr := date.Format("02/01/2006")
		startStr := dateStr + " " + timeText
		start, perr := time.ParseInLocation("02/01/2006 15:04", startStr, loc)
		if perr != nil {
			continue
		}

		duration := parseDurationHM(durationText)
		stop := start.Add(duration)

		prog := domain.Program{
			Channel: channel.XmltvID,
			Source:  "mncvision.id",
			Title:   title,
			Lang:    channel.Lang,
			Start:   start,
			Stop:    stop,
		}

		if ep := parseEpisodeFromTitle(title); ep > 0 {
			prog.Episode = ep
		}

		if s := parseSeasonFromTitle(title); s > 0 {
			prog.Season = s
		}

		programs = append(programs, prog)
	}

	return programs, nil
}

func findRows(n *html.Node) []*html.Node {
	var rows []*html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "tr" {
			if getAttr(node, "valign") == "top" {
				rows = append(rows, node)
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return rows
}

func findCells(n *html.Node) []*html.Node {
	var cells []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "td" {
			cells = append(cells, c)
		}
	}
	return cells
}

func extractLinkText(n *html.Node) string {
	var walk func(*html.Node) string
	walk = func(node *html.Node) string {
		if node.Type == html.ElementNode && node.Data == "a" {
			return strings.TrimSpace(textContent(node))
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if result := walk(c); result != "" {
				return result
			}
		}
		return ""
	}
	return walk(n)
}

func textContent(n *html.Node) string {
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

var durationRegex = regexp.MustCompile(`(\d{2}):(\d{2})`)

func parseDurationHM(s string) time.Duration {
	matches := durationRegex.FindStringSubmatch(s)
	if matches == nil {
		return 0
	}
	h, _ := strconv.Atoi(matches[1])
	m, _ := strconv.Atoi(matches[2])
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute
}

var epRegex = regexp.MustCompile(`Ep\s*(\d+)`)
var seasonRegex = regexp.MustCompile(`S(\d+)`)

func parseEpisodeFromTitle(title string) int {
	matches := epRegex.FindStringSubmatch(title)
	if matches == nil {
		return 0
	}
	n, _ := strconv.Atoi(matches[1])
	return n
}

func parseSeasonFromTitle(title string) int {
	matches := seasonRegex.FindStringSubmatch(title)
	if matches == nil {
		return 0
	}
	n, _ := strconv.Atoi(matches[1])
	return n
}
