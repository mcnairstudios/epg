package startimes

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
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

//go:embed channels.json
var channelsData []byte

var seasonEpisodeRe = regexp.MustCompile(` S(\d+) E(\d+)`)

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

func (p *provider) Name() string        { return "startimestv.com" }
func (p *provider) Description() string { return "StarTimes TV Africa guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "startimestv.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://www.startimestv.com/channeldetail/%s/%s.html",
		channel.SiteID, date.Format("2006-01-02"))

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
		return nil, fmt.Errorf("HTTP %d from startimes", resp.StatusCode)
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
		return []domain.Program{}, nil
	}

	masks := findAllByClass(doc, "mask")
	if len(masks) == 0 {
		return []domain.Program{}, nil
	}

	dateStr := date.Format("2006-01-02")
	var programs []domain.Program

	for _, mask := range masks {
		h4 := findFirstElement(mask, "h4")
		if h4 == nil {
			continue
		}
		h4Text := cleanText(extractText(h4))
		if h4Text == "" {
			continue
		}

		spaceIdx := strings.Index(h4Text, " ")
		if spaceIdx < 0 {
			continue
		}

		timeRange := h4Text[:spaceIdx]
		title := h4Text[spaceIdx+1:]

		parts := strings.SplitN(timeRange, "-", 2)
		if len(parts) != 2 {
			continue
		}

		start, err := time.Parse("2006-01-02 15:04", dateStr+" "+parts[0])
		if err != nil {
			continue
		}
		stop, err := time.Parse("2006-01-02 15:04", dateStr+" "+parts[1])
		if err != nil {
			continue
		}

		p := findFirstElement(mask, "p")
		desc := ""
		if p != nil {
			desc = cleanText(extractText(p))
			if desc == "NA" {
				desc = ""
			}
		}

		var season, episode int
		if matches := seasonEpisodeRe.FindStringSubmatch(title); matches != nil {
			season, _ = strconv.Atoi(matches[1])
			episode, _ = strconv.Atoi(matches[2])
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "startimestv.com",
			Title:       title,
			Description: desc,
			Season:      season,
			Episode:     episode,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\t", "")
	s = strings.ReplaceAll(s, "\n", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

func findAllByClass(n *html.Node, class string) []*html.Node {
	var result []*html.Node
	if n.Type == html.ElementNode {
		cls := getAttr(n, "class")
		for _, c := range strings.Fields(cls) {
			if c == class {
				result = append(result, n)
				break
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result = append(result, findAllByClass(c, class)...)
	}
	return result
}

func findFirstElement(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findFirstElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func extractText(n *html.Node) string {
	if n == nil {
		return ""
	}
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
