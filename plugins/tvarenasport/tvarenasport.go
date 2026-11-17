package tvarenasport

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
	"github.com/gavinmcnair/epg/internal/httpcache"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

//go:embed channels.json
var channelsData []byte

var belgradeTZ, _ = time.LoadLocation("Europe/Belgrade")

var channelLogoRe = regexp.MustCompile(`chanel-([\w-]+?)\.png`)

func init() {
	plugin.Register(&provider{client: httpcache.Wrap(httpclient.Browser())})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "tvarenasport.com" }
func (p *provider) Description() string { return "TV Arena Sport Serbia schedule" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvarenasport.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	req, err := http.NewRequest(http.MethodGet, "https://www.tvarenasport.com/tv-scheme", nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from tvarenasport", resp.StatusCode)
	}

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
		return nil, err
	}

	expectedDate := date.Format("2006-01-02")
	channelContainers := findNodesByClass(doc, "tv-scheme-chanel")

	for _, container := range channelContainers {
		header := findNodeByClass(container, "tv-scheme-chanel-header")
		if header == nil {
			continue
		}

		img := findElement(header, "img")
		if img == nil {
			continue
		}

		src := getAttr(img, "src")
		matches := channelLogoRe.FindStringSubmatch(src)
		if len(matches) < 2 || matches[1] != channel.SiteID {
			continue
		}

		daysContainer := findNodeByClass(container, "tv-scheme-days")
		if daysContainer == nil {
			continue
		}

		var dates []time.Time
		links := findElements(daysContainer, "a")
		for _, link := range links {
			spans := findElements(link, "span")
			if len(spans) >= 3 {
				dateText := strings.TrimSpace(textContent(spans[2]))
				if !strings.HasSuffix(dateText, ".") {
						dateText += "."
					}
					parsed, err := time.Parse("02.01.2006", dateText+fmt.Sprintf("%d", date.Year()))
				if err == nil {
					dates = append(dates, parsed)
				}
			}
		}

		startIdx := -1
		for i, d := range dates {
			if d.Format("2006-01-02") == expectedDate {
				startIdx = i
				break
			}
		}
		if startIdx == -1 {
			continue
		}

		sliders := findNodesByClass(container, "tv-scheme-new-slider-item")
		if startIdx >= len(sliders) {
			continue
		}

		slider := sliders[startIdx]
		entries := parseSchedules(slider, dates[startIdx])

		for i := range entries {
			if i < len(entries)-1 {
				entries[i].Stop = entries[i+1].Start
			} else {
				endOfDay, _ := time.ParseInLocation("2006-01-02 15:04", expectedDate+" 23:59", belgradeTZ)
				entries[i].Stop = endOfDay.UTC()
			}
		}

		programs := make([]domain.Program, 0, len(entries))
		for _, e := range entries {
			programs = append(programs, domain.Program{
				Channel:  channel.XmltvID,
				Source:   "tvarenasport.com",
				Title:    e.title,
				Category: e.category,
				Lang:     channel.Lang,
				Start:    e.Start,
				Stop:     e.Stop,
			})
		}

		return programs, nil
	}

	return []domain.Program{}, nil
}

type scheduleEntry struct {
	title    string
	category string
	Start    time.Time
	Stop     time.Time
}

func parseSchedules(slider *html.Node, date time.Time) []scheduleEntry {
	contents := findNodesByClass(slider, "slider-content")
	entries := make([]scheduleEntry, 0, len(contents))
	for _, content := range contents {
		entries = append(entries, parseSchedule(content, date))
	}
	return entries
}

func parseSchedule(s *html.Node, date time.Time) scheduleEntry {
	topSpan := findNodeByClass(s, "slider-content-top")
	timeText := ""
	if topSpan != nil {
		span := findElement(topSpan, "span")
		if span != nil {
			timeText = strings.TrimSpace(textContent(span))
		}
	}

	start, _ := time.ParseInLocation("2006-01-02 15:04", date.Format("2006-01-02")+" "+timeText, belgradeTZ)

	middleSpan := findNodeByClass(s, "slider-content-middle")
	sport := ""
	if middleSpan != nil {
		span := findElement(middleSpan, "span")
		if span != nil {
			sport = strings.TrimSpace(textContent(span))
		}
	}

	bottomDiv := findNodeByClass(s, "slider-content-bottom")
	titleText := ""
	league := ""
	isLive := false
	if bottomDiv != nil {
		p := findElement(bottomDiv, "p")
		if p != nil {
			titleText = strings.TrimSpace(textContent(p))
		}

		spans := findElements(bottomDiv, "span")
		for _, span := range spans {
			cls := getAttr(span, "class")
			if containsClass(cls, "blob-text") {
				if strings.ToLower(strings.TrimSpace(textContent(span))) == "uživo" {
					isLive = true
				}
			} else if !containsClass(cls, "live-title") && !containsClass(cls, "blob-text") && !containsClass(cls, "blob-border") && !containsClass(cls, "blob") {
				if league == "" {
					league = strings.TrimSpace(textContent(span))
				}
			}
		}
	}

	title := ""
	if isLive {
		title = "(Uživo) "
	}
	if league != "" {
		title += league + ": " + titleText
	} else {
		title += titleText
	}

	return scheduleEntry{
		title:    title,
		category: sport,
		Start:    start.UTC(),
	}
}

func findNodesByClass(n *html.Node, class string) []*html.Node {
	var nodes []*html.Node
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && containsClass(a.Val, class) {
				nodes = append(nodes, n)
				break
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		nodes = append(nodes, findNodesByClass(c, class)...)
	}
	return nodes
}

func findNodeByClass(n *html.Node, class string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && containsClass(a.Val, class) {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findNodeByClass(c, class); found != nil {
			return found
		}
	}
	return nil
}

func findElement(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func findElements(n *html.Node, tag string) []*html.Node {
	var nodes []*html.Node
	if n.Type == html.ElementNode && n.Data == tag {
		nodes = append(nodes, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		nodes = append(nodes, findElements(c, tag)...)
	}
	return nodes
}

func containsClass(classes string, target string) bool {
	for _, c := range strings.Fields(classes) {
		if c == target {
			return true
		}
	}
	return false
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
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

func (p *provider) Timezone() string { return "Europe/Belgrade" }
