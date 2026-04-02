package tvcesoir

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
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

var rome *time.Location

//go:embed channels.json
var channelsData []byte

var timeRe = regexp.MustCompile(`(\d{1,2})h(\d{2})`)

func init() {
	var err error
	rome, err = time.LoadLocation("Europe/Rome")
	if err != nil {
		panic(err)
	}
	plugin.Register(&provider{client: httpclient.Browser()})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "tvcesoir.fr" }
func (p *provider) Description() string { return "French TV program guide tvcesoir.fr" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvcesoir.fr")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://www.tvcesoir.fr/programme-tv/programme/chaine/%s.html?dt=%s",
		channel.SiteID, date.Format("2006-01-02"),
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
		return nil, fmt.Errorf("HTTP %d from tvcesoir", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return []domain.Program{}, nil
	}

	rows := findTableRows(doc)
	programs := make([]domain.Program, 0, len(rows))

	for _, row := range rows {
		cells := getChildElements(row, "td")
		if len(cells) < 2 {
			continue
		}

		timeStr := strings.TrimSpace(textContent(cells[0]))
		title := strings.TrimSpace(textContent(cells[1]))

		if timeStr == "" || title == "" {
			continue
		}

		m := timeRe.FindStringSubmatch(timeStr)
		if m == nil {
			continue
		}

		dateStr := date.Format("2006-01-02")
		start, err := time.ParseInLocation("2006-01-02 15h04", fmt.Sprintf("%s %s", dateStr, timeStr), rome)
		if err != nil {
			continue
		}
		start = start.UTC()

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				start = start.Add(24 * time.Hour)
				date = date.Add(24 * time.Hour)
			}
			prev.Stop = start
		}

		stop := start.Add(30 * time.Minute)

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Lang:    channel.Lang,
			Source:  "tvcesoir.fr",
			Title:   title,
			Start:   start,
			Stop:    stop,
		})
	}

	return programs, nil
}

func findTableRows(n *html.Node) []*html.Node {
	var results []*html.Node
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
		tbody = table
	}

	for c := tbody.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "tr" {
			results = append(results, c)
		}
	}
	return results
}

func getChildElements(n *html.Node, tag string) []*html.Node {
	var results []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			results = append(results, c)
		}
	}
	return results
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

func (p *provider) Timezone() string { return "Europe/Paris" }
