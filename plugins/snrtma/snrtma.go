package snrtma

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
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

var (
	casablanca *time.Location
	dateRe     = regexp.MustCompile(`\d{8}`)
	timeRe     = regexp.MustCompile(`(\d{1,2})H(\d{2})`)
)

func init() {
	casablanca, _ = time.LoadLocation("Africa/Casablanca")
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

func (p *provider) Name() string        { return "snrt.ma" }
func (p *provider) Description() string { return "SNRT Morocco TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "snrt.ma")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://www.snrt.ma/ar/node/%s", channel.SiteID)

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

	lines := findAllByClass(doc, "grille-line")
	if len(lines) == 0 {
		return []domain.Program{}, nil
	}

	var entries []entry

	for _, line := range lines {
		cls := getAttr(line, "class")
		dateMatch := dateRe.FindString(cls)
		if dateMatch == "" {
			continue
		}

		timeNode := findByClass(line, "grille-time")
		if timeNode == nil {
			continue
		}
		timeText := strings.TrimSpace(extractText(timeNode))
		timeMatch := timeRe.FindStringSubmatch(timeText)
		if timeMatch == nil {
			continue
		}

		var h, m int
		fmt.Sscanf(timeMatch[1], "%d", &h)
		fmt.Sscanf(timeMatch[2], "%d", &m)

		var y, mo, d int
		fmt.Sscanf(dateMatch[:4], "%d", &y)
		fmt.Sscanf(dateMatch[4:6], "%d", &mo)
		fmt.Sscanf(dateMatch[6:8], "%d", &d)

		start := time.Date(y, time.Month(mo), d, h, m, 0, 0, casablanca).UTC()

		titleNode := findByClass(line, "program-title-sm")
		title := ""
		if titleNode != nil {
			title = strings.TrimSpace(extractText(titleNode))
		}
		if title == "" {
			continue
		}

		descNode := findByClass(line, "program-description-sm")
		description := ""
		if descNode != nil {
			description = strings.TrimSpace(extractText(descNode))
		}

		catNode := findByClass(line, "genre-first")
		category := ""
		if catNode != nil {
			category = strings.TrimSpace(extractText(catNode))
		}

		entries = append(entries, entry{
			title:       title,
			description: description,
			category:    category,
			start:       start,
		})
	}

	if len(entries) == 0 {
		return []domain.Program{}, nil
	}

	sortEntries(entries)

	nextDay := time.Date(date.Year(), date.Month(), date.Day()+1, 0, 0, 0, 0, casablanca).UTC()
	localDate := date.In(casablanca)

	var programs []domain.Program
	for i, e := range entries {
		var stop time.Time
		if i < len(entries)-1 {
			stop = entries[i+1].start
		} else {
			stop = nextDay
		}

		startLocal := e.start.In(casablanca)
		if startLocal.Year() != localDate.Year() || startLocal.Month() != localDate.Month() || startLocal.Day() != localDate.Day() {
			continue
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "snrt.ma",
			Title:       e.title,
			Description: e.description,
			Category:    e.category,
			Lang:        channel.Lang,
			Start:       e.start,
			Stop:        stop,
		})
	}

	return programs, nil
}

type entry struct {
	title       string
	description string
	category    string
	start       time.Time
}

func sortEntries(entries []entry) {
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].start.Before(entries[j-1].start); j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
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
