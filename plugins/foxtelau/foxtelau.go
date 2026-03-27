package foxtelau

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

func init() {
	plugin.Register(&provider{
		client: http.DefaultClient,
	})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "foxtel.com.au" }
func (p *provider) Description() string { return "Foxtel Australia TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "foxtel-ch9syd", Name: "Channel 9 Sydney", Source: "foxtel.com.au", SiteID: "Channel-9-Sydney/NIN", Lang: "en", XmltvID: "Channel9Sydney.au"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()
	url := fmt.Sprintf("https://www.foxtel.com.au/tv-guide/channel/%s/%s", channel.SiteID, d.Format("2006/01/02"))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept-Language", "en-US,en;")
	req.Header.Set("Cookie", "AAMC_foxtel_0=REGION|7")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

var seasonRe = regexp.MustCompile(`Season: (\d+)`)
var episodeRe = regexp.MustCompile(`Episode: (\d+)`)
var ratingRe = regexp.MustCompile(`\(([^)]+)\)$`)

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return []domain.Program{}, nil
	}

	container := findNodeByID(doc, "epg-channel-events")
	if container == nil {
		return []domain.Program{}, nil
	}

	links := findElements(container, "a")
	programs := make([]domain.Program, 0)

	for _, link := range links {
		scheduledDate := findDataAttr(link, "scheduled-date")
		if scheduledDate == "" {
			continue
		}

		unix, err := strconv.ParseInt(scheduledDate, 10, 64)
		if err != nil {
			continue
		}
		start := time.UnixMilli(unix).UTC()

		descDiv := findNodeByClass(link, "epg-event-description")
		title := ""
		subTitle := ""
		var season, episode int
		ratingStr := ""

		if descDiv != nil {
			title = getDirectText(descDiv)

			innerDiv := findDirectElementByTag(descDiv, "div")
			if innerDiv != nil {
				innerText := strings.TrimSpace(collectText(innerDiv))

				abbrNodes := findElements(innerDiv, "abbr")
				for _, abbr := range abbrNodes {
					titleAttr := getAttr(abbr, "title")
					if m := seasonRe.FindStringSubmatch(titleAttr); m != nil {
						season, _ = strconv.Atoi(m[1])
					}
					if m := episodeRe.FindStringSubmatch(titleAttr); m != nil {
						episode, _ = strconv.Atoi(m[1])
					}
				}

				if m := ratingRe.FindStringSubmatch(innerText); m != nil {
					ratingStr = m[1]
				}

				parts := strings.Split(getDirectText(innerDiv), ",")
				if len(parts) > 0 {
					last := strings.TrimSpace(parts[len(parts)-1])
					last = strings.TrimSuffix(last, "("+ratingStr+")")
					subTitle = strings.TrimSpace(last)
				}
			}
		}

		thumbDiv := findNodeByClass(link, "epg-event-thumbnail")
		image := ""
		if thumbDiv != nil {
			imgs := findElements(thumbDiv, "img")
			if len(imgs) > 0 {
				image = getAttr(imgs[0], "src")
			}
		}

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				start = start.Add(24 * time.Hour)
			}
			prev.Stop = start
		}

		stop := start.Add(30 * time.Minute)

		p := domain.Program{
			Channel:  channel.XmltvID,
			Source:   "foxtel.com.au",
			Title:    title,
			SubTitle: subTitle,
			Image:    image,
			Season:   season,
			Episode:  episode,
			Lang:     channel.Lang,
			Start:    start,
			Stop:     stop,
		}

		if ratingStr != "" {
			p.Category = ratingStr
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func findNodeByID(n *html.Node, id string) *html.Node {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == id {
				return n
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findNodeByID(c, id); result != nil {
			return result
		}
	}
	return nil
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

func findElements(n *html.Node, tag string) []*html.Node {
	var results []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			results = append(results, c)
		}
		results = append(results, findElements(c, tag)...)
	}
	return results
}

func findDirectElementByTag(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			return c
		}
	}
	return nil
}

func findDataAttr(n *html.Node, name string) string {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "data-"+name {
				return a.Val
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if v := findDataAttr(c, name); v != "" {
			return v
		}
	}
	return ""
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func getDirectText(n *html.Node) string {
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		}
	}
	return strings.TrimSpace(sb.String())
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
