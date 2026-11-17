package tvarenasporthr

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

var budapestTZ, _ = time.LoadLocation("Europe/Budapest")

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

func (p *provider) Name() string        { return "tvarenasport.hr" }
func (p *provider) Description() string { return "TV Arena Sport Croatia schedule" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvarenasport.hr")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	req, err := http.NewRequest(http.MethodGet, "https://tvarenaprogram.com/live/v2/hr", nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from tvarenasporthr", resp.StatusCode)
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
	imgSelector := fmt.Sprintf("chanel-%s.png", channel.SiteID)

	var channelSection *html.Node
	imgs := findElements(doc, "img")
	for _, img := range imgs {
		src := getAttr(img, "src")
		if strings.Contains(src, imgSelector) {
			n := img
			for n != nil {
				if n.Type == html.ElementNode && containsClass(getAttr(n, "class"), "tv-scheme-chanel") {
					channelSection = n
					break
				}
				n = n.Parent
			}
			break
		}
	}

	if channelSection == nil {
		return []domain.Program{}, nil
	}

	var dates []time.Time
	daysDiv := findNodeByClass(channelSection, "tv-scheme-days")
	if daysDiv != nil {
		links := findElements(daysDiv, "a")
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
	}

	var allPrograms []domain.Program
	sliderWrappers := findNodesByClass(channelSection, "tv-scheme-new-slider-wrapper")
	sliderIdx := 0
	for _, wrapper := range sliderWrappers {
		items := findNodesByClass(wrapper, "tv-scheme-new-slider-item")
		for i, item := range items {
			if sliderIdx+i < len(dates) {
				entries := parseSchedules(item, dates[sliderIdx+i])
				allPrograms = append(allPrograms, entries...)
			}
		}
		sliderIdx += len(items)
	}

	for i := range allPrograms {
		if i < len(allPrograms)-2 {
			allPrograms[i].Stop = allPrograms[i+1].Start
		} else {
			allPrograms[i].Stop = time.Date(
				allPrograms[i].Start.Year(), allPrograms[i].Start.Month(), allPrograms[i].Start.Day()+1,
				0, 0, 0, 0, time.UTC,
			)
		}
	}

	var filtered []domain.Program
	for _, p := range allPrograms {
		startDate := p.Start.Format("2006-01-02")
		stopDate := p.Stop.Format("2006-01-02")
		if startDate == expectedDate || stopDate == expectedDate {
			p.Channel = channel.XmltvID
			p.Source = "tvarenasport.hr"
			p.Lang = channel.Lang
			filtered = append(filtered, p)
		}
	}

	if filtered == nil {
		return []domain.Program{}, nil
	}
	return filtered, nil
}

func parseSchedules(slider *html.Node, date time.Time) []domain.Program {
	contents := findNodesByClass(slider, "slider-content")
	programs := make([]domain.Program, 0, len(contents))
	for _, content := range contents {
		programs = append(programs, parseSchedule(content, date))
	}
	return programs
}

func parseSchedule(s *html.Node, date time.Time) domain.Program {
	topSpan := findNodeByClass(s, "slider-content-top")
	timeText := ""
	if topSpan != nil {
		span := findElement(topSpan, "span")
		if span != nil {
			timeText = strings.TrimSpace(textContent(span))
		}
	}

	start, _ := time.ParseInLocation("2006-01-02 15:04", date.Format("2006-01-02")+" "+timeText, budapestTZ)

	middleSpan := findNodeByClass(s, "slider-content-middle")
	category := ""
	if middleSpan != nil {
		span := findElement(middleSpan, "span")
		if span != nil {
			category = strings.TrimSpace(textContent(span))
		}
	}

	bottomDiv := findNodeByClass(s, "slider-content-bottom")
	titleText := ""
	description := ""
	if bottomDiv != nil {
		p := findElement(bottomDiv, "p")
		if p != nil {
			titleText = strings.TrimSpace(textContent(p))
		}
		spans := findElements(bottomDiv, "span")
		if len(spans) > 0 {
			description = strings.TrimSpace(textContent(spans[0]))
		}
	}

	title := titleText
	if description != "" {
		title = description
		description = titleText
	} else {
		description = ""
	}

	return domain.Program{
		Title:       title,
		Description: description,
		Category:    category,
		Start:       start.UTC(),
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

func (p *provider) Timezone() string { return "Europe/Zagreb" }
