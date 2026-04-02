package novasports

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

var athensTZ, _ = time.LoadLocation("Europe/Athens")

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

func (p *provider) Name() string        { return "novasports.gr" }
func (p *provider) Description() string { return "Novasports Greece" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "novasports.gr")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()
	url := fmt.Sprintf(
		"https://www.novasports.gr/wp-admin/admin-ajax.php?action=nova_get_template&template=tv-program/broadcast&dt=%s",
		d.Format("2006-01-02"),
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
		return nil, fmt.Errorf("HTTP %d from novasports", resp.StatusCode)
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

	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return []domain.Program{}, nil
	}

	channelDiv := findChannelDiv(doc, channel.SiteID)
	if channelDiv == nil {
		return []domain.Program{}, nil
	}

	programDivs := findProgramDivs(channelDiv)
	d := date.Truncate(24 * time.Hour).UTC()
	programs := make([]domain.Program, 0)

	for _, div := range programDivs {
		timeText := extractByClass(div, "time")
		title := extractByClass(div, "title")
		desc := extractByClass(div, "subtitle")

		if timeText == "" {
			continue
		}

		start, err := time.ParseInLocation("2006-01-02 15:04", d.Format("2006-01-02")+" "+timeText, athensTZ)
		if err != nil {
			continue
		}

		stop := start.Add(30 * time.Minute)

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				d = d.Add(24 * time.Hour)
				start, _ = time.ParseInLocation("2006-01-02 15:04", d.Format("2006-01-02")+" "+timeText, athensTZ)
				stop = start.Add(30 * time.Minute)
			}
			prev.Stop = start.UTC()
		}

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "novasports.gr",
			Title:       title,
			Description: desc,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func findChannelDiv(n *html.Node, channelName string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && strings.Contains(a.Val, "channelist-wrapper") {
				channelNodes := findNodesByClass(n, "channel")
				for _, cn := range channelNodes {
					text := strings.TrimSpace(collectText(cn))
					if text == channelName {
						return n
					}
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findChannelDiv(c, channelName); result != nil {
			return result
		}
	}
	return nil
}

func findProgramDivs(n *html.Node) []*html.Node {
	programContainer := findNodeByClass(n, "channel-program")
	if programContainer == nil {
		return nil
	}
	var divs []*html.Node
	for c := programContainer.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "div" {
			divs = append(divs, c)
		}
	}
	return divs
}

func extractByClass(n *html.Node, class string) string {
	nodes := findNodesByClass(n, class)
	if len(nodes) > 0 {
		return strings.TrimSpace(collectText(nodes[0]))
	}
	return ""
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
		if result := findNodeByClass(c, class); result != nil {
			return result
		}
	}
	return nil
}

func findNodesByClass(n *html.Node, class string) []*html.Node {
	var results []*html.Node
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && strings.Contains(a.Val, class) {
				results = append(results, n)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		results = append(results, findNodesByClass(c, class)...)
	}
	return results
}

func collectText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(collectText(c))
	}
	return sb.String()
}

func (p *provider) Timezone() string { return "Europe/Athens" }
