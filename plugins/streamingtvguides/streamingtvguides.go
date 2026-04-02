package streamingtvguides

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	_ "embed"

	"golang.org/x/net/html"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

var pstLocation *time.Location

func init() {
	var err error
	pstLocation, err = time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic(err)
	}

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

func (p *provider) Name() string        { return "streamingtvguides.com" }
func (p *provider) Description() string { return "Streaming TV Guides channel schedules" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "streamingtvguides.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://streamingtvguides.com/Channel/%s", channel.SiteID)

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
		return nil, fmt.Errorf("HTTP %d from streamingtvguides", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

type containerData struct {
	title       string
	description string
	dateText    string
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, nil
	}

	containers := extractContainerData(doc)
	targetDay := date.Truncate(24 * time.Hour).UTC()

	var programs []domain.Program

	for _, c := range containers {
		start, stop, err := parseTimeString(c.dateText)
		if err != nil {
			continue
		}

		startUTC := start.UTC()
		y, m, d := startUTC.Date()
		startDay := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
		if !startDay.Equal(targetDay) {
			continue
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "streamingtvguides.com",
			Title:       c.title,
			Description: c.description,
			Lang:        channel.Lang,
			Start:       startUTC,
			Stop:        stop.UTC(),
		})
	}

	sort.Slice(programs, func(i, j int) bool {
		return programs[i].Start.Before(programs[j].Start)
	})

	return programs, nil
}

func extractContainerData(doc *html.Node) []containerData {
	var result []containerData
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "container") {
			cd := containerData{}
			extractFromContainer(n, &cd)
			result = append(result, cd)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return result
}

func extractFromContainer(n *html.Node, cd *containerData) {
	if n.Type == html.ElementNode && n.Data == "h5" && hasClass(n, "card-title") {
		cd.title = strings.TrimSpace(directText(n))
		return
	}
	if n.Type == html.ElementNode && n.Data == "p" && hasClass(n, "card-text") {
		cd.description = strings.TrimSpace(directText(n))
		return
	}
	if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "card-body") {
		cd.dateText = strings.TrimSpace(directText(n))
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractFromContainer(c, cd)
	}
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

func parseTimeString(dateStr string) (time.Time, time.Time, error) {
	parts := strings.SplitN(dateStr, " - ", 2)
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid date format: %q", dateStr)
	}

	startStr := strings.TrimSuffix(strings.TrimSpace(parts[0]), " PST")
	stopStr := strings.TrimSuffix(strings.TrimSpace(parts[1]), " PST")

	const layout = "2006-01-02 15:04:05"
	start, err := time.ParseInLocation(layout, startStr, pstLocation)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse start: %w", err)
	}

	stop, err := time.ParseInLocation(layout, stopStr, pstLocation)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse stop: %w", err)
	}

	return start, stop, nil
}

func directText(n *html.Node) string {
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		}
	}
	return sb.String()
}

func (p *provider) Timezone() string { return "America/Los_Angeles" }
