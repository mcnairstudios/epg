package movistarplus

import (
	"bytes"
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

func init() {
	plugin.Register(&provider{client: httpclient.Browser()})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "movistarplus.es" }
func (p *provider) Description() string { return "Movistar Plus+ Spain TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "movistarplus.es")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://www.movistarplus.es/programacion-tv/%s/%s",
		channel.SiteID,
		date.Format("2006-01-02"),
	)

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
		return nil, fmt.Errorf("HTTP %d from movistarplus", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	entries := extractProgramEntries(data)
	if len(entries) == 0 {
		return nil, nil
	}

	loc, _ := time.LoadLocation("Europe/Madrid")
	dateStr := date.Format("2006-01-02")

	programs := make([]domain.Program, 0, len(entries))
	for i, entry := range entries {
		if entry.title == "" || entry.timeText == "" {
			continue
		}

		hours, minutes, ok := parseTimeText(entry.timeText)
		if !ok {
			continue
		}

		startDate, _ := time.ParseInLocation("2006-01-02 15:04", fmt.Sprintf("%s %02d:%02d", dateStr, hours, minutes), loc)
		if hours < 5 {
			startDate = startDate.Add(24 * time.Hour)
		}

		var stopDate time.Time
		if i < len(entries)-1 {
			next := entries[i+1]
			nh, nm, nok := parseTimeText(next.timeText)
			if nok {
				stopDate, _ = time.ParseInLocation("2006-01-02 15:04", fmt.Sprintf("%s %02d:%02d", dateStr, nh, nm), loc)
				if nh < 5 {
					stopDate = stopDate.Add(24 * time.Hour)
				}
				if !stopDate.After(startDate) {
					stopDate = stopDate.Add(24 * time.Hour)
				}
			}
		}

		if stopDate.IsZero() {
			next := startDate.Add(24 * time.Hour)
			stopDate = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, loc)
		}

		prog := domain.Program{
			Channel: channel.XmltvID,
			Source:  "movistarplus.es",
			Title:   entry.title,
			Lang:    channel.Lang,
			Start:   startDate,
			Stop:    stopDate,
		}

		programs = append(programs, prog)
	}

	return programs, nil
}

type programEntry struct {
	title    string
	timeText string
}

var eleIDRegex = regexp.MustCompile(`^ele-\d+$`)

func extractProgramEntries(data []byte) []programEntry {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil
	}

	var entries []programEntry
	var findDivs func(*html.Node)
	findDivs = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			id := getAttr(n, "id")
			if eleIDRegex.MatchString(id) {
				entry := extractEntry(n)
				if entry.title != "" {
					entries = append(entries, entry)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findDivs(c)
		}
	}
	findDivs(doc)

	return entries
}

func extractEntry(n *html.Node) programEntry {
	var entry programEntry
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "li" {
			class := getAttr(node, "class")
			if strings.Contains(class, "title") {
				entry.title = strings.TrimSpace(textContent(node))
			}
			if strings.Contains(class, "time") {
				entry.timeText = strings.TrimSpace(textContent(node))
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return entry
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

func parseTimeText(s string) (int, int, bool) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, false
	}
	m, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, false
	}
	return h, m, true
}

func ParseProgramDescription(data []byte) string {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return ""
	}

	var desc string
	var inShowContent bool
	var inText bool
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if desc != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "div" {
			class := getAttr(n, "class")
			if strings.Contains(class, "show-content") {
				inShowContent = true
			}
			if inShowContent && strings.Contains(class, "text") {
				inText = true
			}
		}
		if inText && n.Type == html.ElementNode && n.Data == "p" {
			text := strings.TrimSpace(textContent(n))
			if text != "" {
				desc = text
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode && n.Data == "div" {
			class := getAttr(n, "class")
			if strings.Contains(class, "text") {
				inText = false
			}
			if strings.Contains(class, "show-content") {
				inShowContent = false
			}
		}
	}
	walk(doc)

	return desc
}

func (p *provider) Timezone() string { return "Europe/Madrid" }
