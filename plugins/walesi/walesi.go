package walesi

import (
	"encoding/json"
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

func (p *provider) Name() string        { return "walesi.com.fj" }
func (p *provider) Description() string { return "Walesi Fiji TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "walesi.com.fj")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := "https://www.walesi.com.fj/wp-admin/admin-ajax.php"

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	var response jsonResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	if response.HTML == "" {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(response.HTML))
	if err != nil {
		return []domain.Program{}, nil
	}

	rows := findTableRows(doc)
	if len(rows) == 0 {
		return []domain.Program{}, nil
	}

	loc, err := time.LoadLocation("Pacific/Fiji")
	if err != nil {
		return nil, err
	}

	programs := make([]domain.Program, 0, len(rows))
	for _, row := range rows {
		timeStr, title := extractRowData(row)
		if timeStr == "" || title == "" {
			continue
		}

		fullTime := fmt.Sprintf("%s %s", date.Format("2006-01-02"), timeStr)
		start, err := time.ParseInLocation("2006-01-02 3:04 pm", fullTime, loc)
		if err != nil {
			continue
		}

		stop := start.Add(30 * time.Minute)

		if len(programs) > 0 {
			programs[len(programs)-1].Stop = start.UTC()
		}

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "walesi.com.fj",
			Title:   title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		}
		programs = append(programs, p)
	}

	return programs, nil
}

type jsonResponse struct {
	HTML string `json:"html"`
}

func findTableRows(doc *html.Node) []*html.Node {
	var rows []*html.Node
	var tbody *html.Node

	var findTbody func(*html.Node)
	findTbody = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tbody" {
			tbody = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findTbody(c)
		}
	}
	findTbody(doc)

	if tbody == nil {
		return nil
	}

	for c := tbody.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "tr" {
			rows = append(rows, c)
		}
	}
	return rows
}

func extractRowData(row *html.Node) (timeStr, title string) {
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "td" {
			if hasClass(n, "extvs-table1-time") {
				timeStr = strings.TrimSpace(getSpanText(n))
			}
			if hasClass(n, "extvs-table1-programme") {
				title = strings.TrimSpace(getH3Text(n))
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(row)
	return
}

func getSpanText(n *html.Node) string {
	var result string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "span" {
			result = textContent(n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return result
}

func getH3Text(n *html.Node) string {
	var result string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "h3" {
			result = textContent(n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return result
}

func hasClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			for _, c := range strings.Fields(attr.Val) {
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
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textContent(c))
	}
	return sb.String()
}
