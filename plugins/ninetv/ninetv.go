package ninetv

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

var jerusalemTZ *time.Location

func init() {
	var err error
	jerusalemTZ, err = time.LoadLocation("Asia/Jerusalem")
	if err != nil {
		jerusalemTZ = time.FixedZone("Asia/Jerusalem", 2*60*60)
	}
}

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

func (p *provider) Name() string        { return "9tv.co.il" }
func (p *provider) Description() string { return "Channel 9 Israel EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "9tv.co.il")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://www.9tv.co.il/BroadcastSchedule/getBrodcastSchedule?date=%s",
		date.Format("02/01/2006 00:00:00"),
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
		return nil, fmt.Errorf("HTTP %d from ninetv", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

var bgImageRegex = regexp.MustCompile(`url\((.*?)\)`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	liNodes := findAllElements(doc, "li")
	if len(liNodes) == 0 {
		return []domain.Program{}, nil
	}

	dateStr := date.Format("2006-01-02")
	var programs []domain.Program

	for _, li := range liNodes {
		timeNode := findNodeWithClass(li, "div", "guide_list_time")
		if timeNode == nil {
			continue
		}
		timeStr := strings.TrimSpace(extractText(timeNode))
		if timeStr == "" {
			continue
		}

		start, parseErr := time.ParseInLocation("2006-01-02 15:04", fmt.Sprintf("%s %s", dateStr, timeStr), jerusalemTZ)
		if parseErr != nil {
			continue
		}

		titleNode := findNodeWithClass(li, "h3", "guide_info_title")
		title := ""
		if titleNode != nil {
			title = strings.TrimSpace(extractText(titleNode))
		}

		descNode := findDescriptionNode(li)
		desc := ""
		if descNode != nil {
			desc = strings.TrimSpace(extractText(descNode))
		}

		image := ""
		pictNode := findNodeWithClass(li, "div", "guide_info_pict")
		if pictNode != nil {
			style := getAttr(pictNode, "style")
			if style != "" {
				matches := bgImageRegex.FindStringSubmatch(style)
				if matches != nil && matches[1] != "" {
					image = "https://www.9tv.co.il" + matches[1]
				}
			}
		}

		if len(programs) > 0 {
			programs[len(programs)-1].Stop = start.UTC()
		}

		stop := start.Add(1 * time.Hour)

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "9tv.co.il",
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

func findDescriptionNode(li *html.Node) *html.Node {
	txtGroup := findNodeWithClass(li, "div", "guide_txt_group")
	if txtGroup == nil {
		return nil
	}
	for c := txtGroup.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "div" {
			return c
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
		if strings.Contains(getAttr(n, "class"), class) {
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

func findAllElements(n *html.Node, tag string) []*html.Node {
	var results []*html.Node
	if n.Type == html.ElementNode && n.Data == tag {
		results = append(results, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		results = append(results, findAllElements(c, tag)...)
	}
	return results
}

func (p *provider) Timezone() string { return "Asia/Jerusalem" }
