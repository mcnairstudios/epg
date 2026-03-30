package meuguia

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

var saoPauloTZ *time.Location

func init() {
	var err error
	saoPauloTZ, err = time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		saoPauloTZ = time.FixedZone("America/Sao_Paulo", -3*60*60)
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

func (p *provider) Name() string        { return "meuguia.tv" }
func (p *provider) Description() string { return "Meu Guia TV Brazil EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "meuguia.tv")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://meuguia.tv/programacao/canal/%s", channel.SiteID)

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
		return nil, fmt.Errorf("HTTP %d from meuguia", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

var epRegex = regexp.MustCompile(`T(\d+) EP(\d+)`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}

	type rawItem struct {
		title   string
		start   time.Time
		season  int
		episode int
	}

	year := date.Format("2006")
	var items []rawItem
	var lastDate string

	ulNode := findNodeWithClass(doc, "ul", "mw")
	if ulNode == nil {
		return []domain.Program{}, nil
	}

	for li := ulNode.FirstChild; li != nil; li = li.NextSibling {
		if li.Type != html.ElementNode || li.Data != "li" {
			continue
		}

		class := getAttr(li, "class")

		if strings.Contains(class, "subheader") {
			text := strings.TrimSpace(extractText(li))
			parts := strings.SplitN(text, ", ", 2)
			if len(parts) == 2 {
				lastDate = strings.TrimSpace(parts[1])
			}
			continue
		}

		if strings.Contains(class, "divider") || lastDate == "" {
			continue
		}

		aNode := findChild(li, "a")
		if aNode == nil {
			continue
		}

		title := strings.TrimSpace(getAttr(aNode, "title"))
		timeDiv := findChildWithClass(aNode, "div", "time")
		if title == "" || timeDiv == nil {
			continue
		}

		timeStr := strings.TrimSpace(extractText(timeDiv))
		if timeStr == "" {
			continue
		}

		fullTimeStr := fmt.Sprintf("%s/%s %s", lastDate, year, timeStr)
		start, parseErr := time.ParseInLocation("02/01/2006 15:04", fullTimeStr, saoPauloTZ)
		if parseErr != nil {
			continue
		}

		item := rawItem{title: title, start: start.UTC()}
		matches := epRegex.FindStringSubmatch(title)
		if matches != nil {
			item.season, _ = strconv.Atoi(matches[1])
			item.episode, _ = strconv.Atoi(matches[2])
		}
		items = append(items, item)
	}

	dateDay := date.Truncate(24 * time.Hour).UTC()
	if len(items) > 1 {
		for i := 0; i < len(items)-1; i++ {
			items[i].start = items[i].start
		}
	}

	var result []domain.Program
	for i, item := range items {
		if !item.start.Truncate(24 * time.Hour).Equal(dateDay) {
			continue
		}

		prog := domain.Program{
			Channel: channel.XmltvID,
			Source:  "meuguia.tv",
			Title:   item.title,
			Lang:    channel.Lang,
			Start:   item.start,
			Season:  item.season,
			Episode: item.episode,
		}

		if i+1 < len(items) {
			prog.Stop = items[i+1].start
		}

		result = append(result, prog)
	}

	return result, nil
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

func findChild(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			return c
		}
		if found := findChild(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func findChildWithClass(n *html.Node, tag, class string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			if strings.Contains(getAttr(c, "class"), class) {
				return c
			}
		}
		if found := findChildWithClass(c, tag, class); found != nil {
			return found
		}
	}
	return nil
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
