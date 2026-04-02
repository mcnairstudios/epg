package tvmimt

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

func (p *provider) Name() string        { return "tvmi.mt" }
func (p *provider) Description() string { return "TVM Malta TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvmi.mt")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()
	url := fmt.Sprintf("https://tvmi.mt/schedule/%s/%s", channel.SiteID, d.Format("2006-01-02"))

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
		return nil, fmt.Errorf("HTTP %d from tvmimt", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel, date)
}

var timeRe = regexp.MustCompile(`^(\d{2}):(\d{2})`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation("Europe/Malta")
	if err != nil {
		return nil, err
	}

	mt8 := findByClass(doc, "mt-8")
	if mt8 == nil {
		return []domain.Program{}, nil
	}

	items := directChildDivs(mt8)
	programs := make([]domain.Program, 0, len(items))
	currentDate := date

	for _, item := range items {
		ti, title, desc, image := extractItemData(item)
		if ti == "" || title == "" {
			continue
		}

		matches := timeRe.FindStringSubmatch(ti)
		if matches == nil {
			continue
		}

		timeStr := fmt.Sprintf("%s %s:%s", currentDate.Format("2006-01-02"), matches[1], matches[2])
		start, err := time.ParseInLocation("2006-01-02 15:04", timeStr, loc)
		if err != nil {
			continue
		}

		if len(programs) > 0 && start.Before(programs[len(programs)-1].Start) {
			start = start.Add(24 * time.Hour)
			currentDate = currentDate.Add(24 * time.Hour)
		}

		if len(programs) > 0 {
			programs[len(programs)-1].Stop = start.UTC()
		}

		stop := start.Add(30 * time.Minute)

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tvmi.mt",
			Title:       title,
			Description: desc,
			Image:       image,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		}
		programs = append(programs, p)
	}

	return programs, nil
}

func extractItemData(item *html.Node) (timeStr, title, desc, image string) {
	var dataBg string
	var walkForBg func(*html.Node)
	walkForBg = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				if attr.Key == "data-bg" && dataBg == "" {
					dataBg = attr.Val
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkForBg(c)
		}
	}
	walkForBg(item)

	if dataBg != "" {
		if strings.HasPrefix(dataBg, "//") {
			image = "https:" + dataBg
		} else {
			image = dataBg
		}
	}

	flexDiv := findByClass(item, "flex-1")
	if flexDiv == nil {
		return
	}

	children := directChildDivs(flexDiv)
	for i, child := range children {
		text := strings.TrimSpace(textContent(child))
		switch i {
		case 0:
			timeStr = text
		case 1:
			title = text
		case 2:
			desc = text
		}
	}

	return
}

func findByClass(root *html.Node, class string) *html.Node {
	var result *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if result != nil {
			return
		}
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				if attr.Key == "class" {
					for _, c := range strings.Fields(attr.Val) {
						if c == class {
							result = n
							return
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(root)
	return result
}

func directChildDivs(n *html.Node) []*html.Node {
	var divs []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "div" {
			divs = append(divs, c)
		}
	}
	return divs
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

func (p *provider) Timezone() string { return "Europe/Malta" }
