package thesportplus

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

var timezones = map[string]string{
	"usa":  "America/New_York",
	"aus":  "Australia/Sydney",
	"euro": "UTC",
}

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

func (p *provider) Name() string        { return "thesportplus.com" }
func (p *provider) Description() string { return "The Sport Plus TV schedule" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "thesportplus.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	u := fmt.Sprintf(
		"https://www.thesportplus.com/schedule_%s.php?d=%s",
		channel.SiteID,
		date.Format("2006-01-02"),
	)

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from thesportplus", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	items := parseItems(string(data))
	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	tzName := timezones[channel.SiteID]
	if tzName == "" {
		tzName = "UTC"
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		loc = time.UTC
	}

	var programs []domain.Program
	for _, item := range items {
		start := parseTime(item.timeStr, date, loc)
		if start.IsZero() {
			continue
		}

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) && start.Hour() < 12 {
				start = start.Add(24 * time.Hour)
				date = date.Add(24 * time.Hour)
			}
			prev.Stop = start
		}

		stop := start.Add(1 * time.Hour)

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "thesportplus.com",
			Title:       item.title,
			Description: item.description,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type programItem struct {
	timeStr     string
	title       string
	description string
}

func parseItems(content string) []programItem {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil
	}

	var items []programItem
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "resume-item") {
			item := parseResumeItem(n)
			if item.title != "" {
				items = append(items, item)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return items
}

func parseResumeItem(n *html.Node) programItem {
	var item programItem
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		switch c.Data {
		case "h4":
			item.timeStr = strings.TrimSpace(getText(c))
		case "h5":
			item.title = strings.TrimSpace(getText(c))
		case "p":
			item.description = strings.TrimSpace(getText(c))
		}
	}
	return item
}

func parseTime(timeStr string, date time.Time, loc *time.Location) time.Time {
	if timeStr == "" {
		return time.Time{}
	}
	dateStr := date.Format("2006-01-02") + " " + timeStr
	t, err := time.ParseInLocation("2006-01-02 15:04", dateStr, loc)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
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

func getText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(getText(c))
	}
	return sb.String()
}

func (p *provider) Timezone() string { return "UTC" }
