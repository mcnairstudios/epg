package sattv

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/fetchcache"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{
		client: httpclient.Browser(),
		docs:   newDocs(),
	})
}

type provider struct {
	client httpClient
	docs   *fetchcache.Cache[doc]
}

// doc maps a channel title to its listing container in one response page.
type doc map[string]*html.Node

func newDocs() *fetchcache.Cache[doc] {
	return fetchcache.New(fetchcache.Options[doc]{
		TTL:        fetchcache.DocTTL[doc](15 * time.Minute),
		MaxEntries: 4,
	})
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "sat.tv" }
func (p *provider) Description() string { return "Sat.TV satellite TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	chs, err := channels.Load(channelsData, "sat.tv")
	if err != nil {
		return nil, err
	}
	return channels.GroupBy(chs, func(c domain.Channel) string {
		parts := strings.SplitN(c.SiteID, "#", 3)
		if len(parts) < 2 {
			return c.SiteID
		}
		return parts[0] + "#" + parts[1] + "#" + c.Lang
	}), nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := "https://www.sat.tv/wp-content/themes/twentytwenty-child/ajax_chaines.php"

	parts := strings.SplitN(channel.SiteID, "#", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid site_id format: %s", channel.SiteID)
	}

	body := fmt.Sprintf(
		"dateFiltre=%s&hoursFiltre=0&satLineup=%s&satSatellite=%s&userDateTime=%d&userTimezone=Europe/London",
		date.Format("2006-01-02"),
		parts[1],
		parts[0],
		date.UnixMilli(),
	)

	d, err := p.docs.Get(body+"|"+channel.Lang, func() (doc, error) {
		return p.download(url, body, channel.Lang)
	})
	if err != nil {
		return nil, err
	}
	return programsFor(d, channel, parts[2], date), nil
}

func (p *provider) download(url, body, lang string) (doc, error) {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Cookie", fmt.Sprintf("pll_language=%s", lang))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &fetchcache.StatusError{Code: resp.StatusCode, Source: "sattv"}
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return buildDoc(respBody)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	parts := strings.SplitN(channel.SiteID, "#", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid site_id format: %s", channel.SiteID)
	}

	d, err := buildDoc(data)
	if err != nil {
		return nil, err
	}
	return programsFor(d, channel, parts[2], date), nil
}

func buildDoc(data []byte) (doc, error) {
	d := make(doc)
	if len(data) == 0 {
		return d, nil
	}

	root, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	for _, container := range findNodesByClass(root, "container-channel-events") {
		titleNode := findNodeByClass(container, "channel-title")
		if titleNode == nil {
			continue
		}
		name := strings.TrimSpace(textContent(titleNode))
		if _, dup := d[name]; !dup {
			d[name] = container
		}
	}
	return d, nil
}

func programsFor(d doc, channel domain.Channel, channelName string, date time.Time) []domain.Program {
	targetContainer := d[channelName]
	if targetContainer == nil {
		return []domain.Program{}
	}

	events := findNodesByClass(targetContainer, "container-event")
	dateStr := date.Format("2006-01-02")

	timeRe := regexp.MustCompile(`(\d{2}:\d{2})`)
	durationRe := regexp.MustCompile(`(\d{2})h(\d{2})`)

	programs := make([]domain.Program, 0, len(events))
	for _, event := range events {
		titleNode := findNodeByClass(event, "event-data-title")
		descNode := findNodeByClass(event, "event-data-desc")
		dateNode := findNodeByClass(event, "event-data-date")
		infoNode := findNodeByClass(event, "event-data-info")

		if titleNode == nil || dateNode == nil {
			continue
		}

		title := textContent(titleNode)
		description := ""
		if descNode != nil {
			description = textContent(descNode)
		}

		dateText := strings.TrimSpace(textContent(dateNode))
		timeMatch := timeRe.FindStringSubmatch(dateText)
		if len(timeMatch) < 2 {
			continue
		}

		start, err := time.Parse("2006-01-02 15:04", dateStr+" "+timeMatch[1])
		if err != nil {
			continue
		}

		var durationMinutes int
		if infoNode != nil {
			infoText := strings.TrimSpace(textContent(infoNode))
			durMatch := durationRe.FindStringSubmatch(infoText)
			if len(durMatch) >= 3 {
				h, _ := strconv.Atoi(durMatch[1])
				m, _ := strconv.Atoi(durMatch[2])
				durationMinutes = h*60 + m
			}
		}

		stop := start.Add(time.Duration(durationMinutes) * time.Minute)

		image := findEventImage(event)

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "sat.tv",
			Title:       title,
			Description: description,
			Image:       image,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		})
	}

	return programs
}

func findEventImage(event *html.Node) string {
	logoNode := findNodeByClass(event, "event-logo")
	if logoNode == nil {
		return ""
	}

	img := findElement(logoNode, "img")
	if img == nil {
		return ""
	}

	for _, a := range img.Attr {
		if a.Key == "class" && strings.Contains(a.Val, "no-img") {
			return ""
		}
	}

	src := getAttr(img, "src")
	if src == "" {
		return ""
	}

	if strings.HasPrefix(src, "/") {
		return "https://sat.tv" + src
	}
	return src
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

func (p *provider) Timezone() string { return "Africa/Cairo" }
