package digiturk

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

var istanbulTZ *time.Location

//go:embed channels.json
var channelsData []byte

func init() {
	istanbulTZ, _ = time.LoadLocation("Europe/Istanbul")
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

func (p *provider) Name() string        { return "digiturk.com.tr" }
func (p *provider) Description() string { return "Digiturk Turkish satellite TV provider" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "digiturk.com.tr")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	u := fmt.Sprintf(
		"https://www.digiturk.com.tr/Ajax/GetTvGuideFromDigiturk?Day=%s+00%%3A00%%3A00",
		url.QueryEscape(date.Format("01/02/2006")),
	)

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from digiturk", resp.StatusCode)
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

	siteIDRe := regexp.MustCompile(`\s(\d+)\)`)

	var programs []domain.Program
	channelDetails := findNodesByClass(doc, "channelDetail")

	dateStr := date.Format("2006-01-02")

	for _, detail := range channelDetails {
		titleNode := findNodeByClass(detail, "tvGuideResult-box-wholeDates-title")
		if titleNode == nil {
			continue
		}

		onclick := getAttr(titleNode, "onclick")
		if onclick == "" {
			continue
		}

		matches := siteIDRe.FindStringSubmatch(onclick)
		if len(matches) < 2 {
			continue
		}

		if matches[1] != channel.SiteID {
			continue
		}

		title := strings.TrimSpace(textContent(titleNode))

		timeNode := findNodeByClass(detail, "tvGuideResult-box-wholeDates-time-hour")
		durationNode := findNodeByClass(detail, "tvGuideResult-box-wholeDates-time-totalMinute")

		if timeNode == nil || durationNode == nil {
			continue
		}

		startTime := strings.TrimSpace(textContent(timeNode))
		durationText := strings.TrimSpace(textContent(durationNode))

		durationRe := regexp.MustCompile(`\d+`)
		durationMatch := durationRe.FindString(durationText)
		duration, _ := strconv.Atoi(durationMatch)

		start, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+startTime, istanbulTZ)
		if err != nil {
			continue
		}
		stop := start.Add(time.Duration(duration) * time.Minute)

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "digiturk.com.tr",
			Title:   title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		})
	}

	if programs == nil {
		programs = []domain.Program{}
	}

	return programs, nil
}

func findNodesByClass(n *html.Node, class string) []*html.Node {
	var nodes []*html.Node
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && strings.Contains(a.Val, class) {
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
			if a.Key == "class" && strings.Contains(a.Val, class) {
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
