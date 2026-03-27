package sattv2

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

const apiEndpoint = "https://www.sat.tv/wp-content/themes/twentytwenty-child/ajax_chaines.php"

func (p *provider) Name() string        { return "sat.tv.v2" }
func (p *provider) Description() string { return "Sat.TV satellite TV guide (v2)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "sat.tv.v2")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	parts := strings.SplitN(channel.SiteID, "#", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid site_id format: %s", channel.SiteID)
	}
	satSatellite := parts[0]
	satLineup := parts[1]

	params := url.Values{}
	params.Set("dateFiltre", date.Format("2006-01-02"))
	params.Set("hoursFiltre", "0")
	params.Set("satLineup", satLineup)
	params.Set("satSatellite", satSatellite)
	params.Set("userDateTime", fmt.Sprintf("%d", date.UnixMilli()))
	params.Set("userTimezone", "Europe/London")

	req, err := http.NewRequest(http.MethodPost, apiEndpoint, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Cookie", fmt.Sprintf("pll_language=%s", channel.Lang))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

var durationRegex = regexp.MustCompile(`(\d{2})h(\d{2})`)
var timeRegex = regexp.MustCompile(`(\d{2}:\d{2})`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	parts := strings.SplitN(channel.SiteID, "#", 3)
	if len(parts) < 3 {
		return []domain.Program{}, nil
	}
	channelName := parts[2]

	channelContainer := findChannelContainer(doc, channelName)
	if channelContainer == nil {
		return []domain.Program{}, nil
	}

	events := findAllWithClass(channelContainer, "div", "container-event")

	dateStr := date.Format("2006-01-02")
	programs := make([]domain.Program, 0, len(events))
	for _, event := range events {
		titleNode := findNodeWithClass(event, "span", "event-data-title")
		title := ""
		if titleNode != nil {
			title = strings.TrimSpace(extractText(titleNode))
		}

		descNode := findNodeWithClass(event, "span", "event-data-desc")
		desc := ""
		if descNode != nil {
			desc = strings.TrimSpace(extractText(descNode))
		}

		dateNode := findNodeWithClass(event, "span", "event-data-date")
		timeStr := ""
		if dateNode != nil {
			text := strings.TrimSpace(extractText(dateNode))
			matches := timeRegex.FindStringSubmatch(text)
			if matches != nil {
				timeStr = matches[1]
			}
		}

		if timeStr == "" {
			continue
		}

		start, parseErr := time.Parse("2006-01-02 15:04", fmt.Sprintf("%s %s", dateStr, timeStr))
		if parseErr != nil {
			continue
		}

		durationMinutes := 0
		infoNode := findNodeWithClass(event, "span", "event-data-info")
		if infoNode != nil {
			infoText := strings.TrimSpace(extractText(infoNode))
			dMatches := durationRegex.FindStringSubmatch(infoText)
			if dMatches != nil {
				h, _ := strconv.Atoi(dMatches[1])
				m, _ := strconv.Atoi(dMatches[2])
				durationMinutes = h*60 + m
			}
		}

		stop := start.Add(time.Duration(durationMinutes) * time.Minute)

		image := ""
		logoNode := findNodeWithClass(event, "div", "event-logo")
		if logoNode != nil {
			img := findElement(logoNode, "img")
			if img != nil {
				cls := getAttr(img, "class")
				if !strings.Contains(cls, "no-img") {
					src := getAttr(img, "src")
					if src != "" {
						image = "https://sat.tv" + src
					}
				}
			}
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "sat.tv.v2",
			Title:       title,
			Description: desc,
			Image:       image,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		})
	}

	return programs, nil
}

func findChannelContainer(doc *html.Node, channelName string) *html.Node {
	containers := findAllWithClass(doc, "div", "container-channel-events")
	for _, container := range containers {
		titleNode := findNodeWithClass(container, "span", "channel-title")
		if titleNode != nil {
			text := strings.TrimSpace(extractText(titleNode))
			if text == channelName {
				return container
			}
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

func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(extractText(c))
	}
	return sb.String()
}

func findNodeWithClass(n *html.Node, tag, class string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		if hasClass(getAttr(n, "class"), class) {
			return n
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findNodeWithClass(c, tag, class); found != nil {
			return found
		}
	}
	return nil
}

func findAllWithClass(n *html.Node, tag, class string) []*html.Node {
	var results []*html.Node
	if n.Type == html.ElementNode && n.Data == tag {
		if hasClass(getAttr(n, "class"), class) {
			results = append(results, n)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		results = append(results, findAllWithClass(c, tag, class)...)
	}
	return results
}

func hasClass(attrVal, class string) bool {
	for _, c := range strings.Fields(attrVal) {
		if c == class {
			return true
		}
	}
	return false
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
