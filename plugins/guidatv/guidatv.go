package guidatv

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
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

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "guida.tv" }
func (p *provider) Description() string { return "Guida TV Italy EPG" }

var romeLocation *time.Location


func init() {
	romeLocation, _ = time.LoadLocation("Europe/Rome")
}

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "guida.tv")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://www.guida.tv/programmi-tv/palinsesto/canale/%s.html?dt=%s",
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

	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return []domain.Program{}, nil
	}

	rows := findTableRows(doc)
	programs := make([]domain.Program, 0, len(rows))
	currentDate := date

	for i, row := range rows {
		cells := findElements(row, "td")
		if len(cells) < 2 {
			continue
		}

		timeStr := strings.TrimSpace(textContent(cells[0]))
		title := strings.TrimSpace(textContent(cells[1]))
		if timeStr == "" || title == "" {
			continue
		}

		dateStr := fmt.Sprintf("%s %s", currentDate.Format("2006-01-02"), timeStr)
		start, err := time.ParseInLocation("2006-01-02 15:04", dateStr, romeLocation)
		if err != nil {
			continue
		}

		if len(programs) > 0 && start.Before(programs[len(programs)-1].Start) {
			currentDate = currentDate.Add(24 * time.Hour)
			dateStr = fmt.Sprintf("%s %s", currentDate.Format("2006-01-02"), timeStr)
			start, err = time.ParseInLocation("2006-01-02 15:04", dateStr, romeLocation)
			if err != nil {
				continue
			}
		}

		stop := start.Add(30 * time.Minute)

		if len(programs) > 0 {
			programs[len(programs)-1].Stop = start.UTC()
		}

		_ = i

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "guida.tv",
			Title:   title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		})
	}

	return programs, nil
}

func findTableRows(n *html.Node) []*html.Node {
	table := findElement(n, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "table" && hasClass(node, "table")
	})
	if table == nil {
		return nil
	}

	tbody := findElement(table, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "tbody"
	})
	if tbody == nil {
		return nil
	}

	var rows []*html.Node
	for c := tbody.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "tr" {
			rows = append(rows, c)
		}
	}
	return rows
}

func findElements(n *html.Node, tag string) []*html.Node {
	var results []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			results = append(results, c)
		}
	}
	return results
}

func findElement(n *html.Node, match func(*html.Node) bool) *html.Node {
	if match(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findElement(c, match); result != nil {
			return result
		}
	}
	return nil
}

func hasClass(n *html.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range strings.Fields(a.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

func textContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var buf strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		buf.WriteString(textContent(c))
	}
	return buf.String()
}
