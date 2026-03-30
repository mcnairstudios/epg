package sjonvarp

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
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

func (p *provider) Name() string        { return "sjonvarp.is" }
func (p *provider) Description() string { return "Sjonvarp.is Icelandic TV listings" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "sjonvarp.is")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.UTC()
	url := fmt.Sprintf(
		"http://www.sjonvarp.is/index.php?Tm=%%3F&p=idag&c=%s&y=%s&m=%s&d=%s",
		channel.SiteID,
		d.Format("2006"),
		d.Format("01"),
		d.Format("02"),
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
		return nil, fmt.Errorf("HTTP %d from sjonvarp", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, nil
	}

	container := findBySelector(doc, "div", "day-listing")
	if container == nil {
		return nil, nil
	}

	items := findDayListingItems(container)
	programs := make([]domain.Program, 0, len(items))
	currentDate := date.UTC()

	for _, item := range items {
		title := textByClass(item, "day-listing-title")
		description := textByClass(item, "day-listing-description")
		timeStr := textByClass(item, "day-listing-time")

		start := parseStartTime(timeStr, currentDate)
		if start.IsZero() {
			continue
		}

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				currentDate = currentDate.Add(24 * time.Hour)
				start = parseStartTime(timeStr, currentDate)
			}
			prev.Stop = start
		}

		stop := start.Add(30 * time.Minute)

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "sjonvarp.is",
			Title:       title,
			Description: description,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

func parseStartTime(timeStr string, date time.Time) time.Time {
	timeStr = strings.TrimSpace(timeStr)
	if timeStr == "" {
		return time.Time{}
	}
	dateStr := date.Format("2006-01-02")
	t, err := time.Parse("2006-01-02 15:04", dateStr+" "+timeStr)
	if err != nil {
		return time.Time{}
	}
	return t
}

func findBySelector(n *html.Node, tag, class string) *html.Node {
	var result *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if result != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == tag && hasClass(n, class) {
			result = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return result
}

func findDayListingItems(container *html.Node) []*html.Node {
	var items []*html.Node
	for c := container.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "div" && !hasClass(c, "day-listing-channel") {
			items = append(items, c)
		}
	}
	return items
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

func findByClass(n *html.Node, class string) *html.Node {
	var result *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if result != nil {
			return
		}
		if n.Type == html.ElementNode && hasClass(n, class) {
			result = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return result
}

func textByClass(n *html.Node, class string) string {
	el := findByClass(n, class)
	if el == nil {
		return ""
	}
	return strings.TrimSpace(textContent(el))
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
