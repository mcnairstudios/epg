package kvffo

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

func (p *provider) Name() string        { return "kvf.fo" }
func (p *provider) Description() string { return "Kringvarp Foroya Faroese TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "kvf.fo")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()
	url := fmt.Sprintf("https://kvf.fo/nskra/sv?date=%s", d.Format("2006-01-02"))

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
		return nil, fmt.Errorf("HTTP %d from kvffo", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel, date)
}

var timeRe = regexp.MustCompile(`(\d{2}:\d{2})`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation("Atlantic/Faroe")
	if err != nil {
		return nil, err
	}

	rows := findViewsRows(doc)
	programs := make([]domain.Program, 0, len(rows))

	currentDate := date

	for _, row := range rows {
		timeText, title := extractRowData(row)
		if timeText == "" {
			continue
		}

		times := timeRe.FindAllString(timeText, -1)
		if len(times) < 1 {
			continue
		}

		startTime := times[0]
		startStr := fmt.Sprintf("%s %s", currentDate.Format("2006-01-02"), startTime)
		start, err := time.ParseInLocation("2006-01-02 15:04", startStr, loc)
		if err != nil {
			continue
		}

		if len(programs) > 0 && start.Before(programs[len(programs)-1].Stop) {
			start = start.Add(24 * time.Hour)
			currentDate = currentDate.Add(24 * time.Hour)
		}

		var stop time.Time
		if len(times) >= 2 {
			stopTime := times[1]
			stopStr := fmt.Sprintf("%s %s", currentDate.Format("2006-01-02"), stopTime)
			stop, err = time.ParseInLocation("2006-01-02 15:04", stopStr, loc)
			if err != nil {
				stop = start.Add(30 * time.Minute)
			}
			if stop.Before(start) {
				stop = stop.Add(24 * time.Hour)
				currentDate = currentDate.Add(24 * time.Hour)
			}
		} else {
			stop = start.Add(30 * time.Minute)
		}

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "kvf.fo",
			Title:   title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		}
		programs = append(programs, p)
	}

	return programs, nil
}

func findViewsRows(doc *html.Node) []*html.Node {
	var rows []*html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "views-row") {
					rows = append(rows, n)
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return rows
}

func extractRowData(row *html.Node) (string, string) {
	var timeText, title string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			if hasClass(n, "s-time1") {
				timeText = strings.TrimSpace(textContent(n))
			}
			if hasClass(n, "s-heiti") {
				title = strings.TrimSpace(textContent(n))
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(row)
	return timeText, title
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

func (p *provider) Timezone() string { return "Atlantic/Faroe" }
