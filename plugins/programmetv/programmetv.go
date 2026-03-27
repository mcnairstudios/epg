package programmetv

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

var paris *time.Location

func init() {
	var err error
	paris, err = time.LoadLocation("Europe/Paris")
	if err != nil {
		panic(err)
	}

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

func (p *provider) Name() string        { return "programme-tv.net" }
func (p *provider) Description() string { return "Programme TV listings (France)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "programme-tv.net")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.In(paris).Format("2006-01-02")
	url := fmt.Sprintf(
		"https://www.programme-tv.net/programme/chaine/%s/programme-%s.html",
		d, channel.SiteID,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", "authId=b7154156fe4fb8acdb6f38e1207c6231")

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

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	cards := findAllByClass(doc, "mainBroadcastCard")
	programs := make([]domain.Program, 0, len(cards))

	for _, card := range cards {
		title := textByClass(card, "mainBroadcastCard-title")
		subTitle := textByClass(card, "mainBroadcastCard-subtitle")
		category := textByClass(card, "mainBroadcastCard-format")
		image := parseImage(card)
		start := parseStart(card, date)
		duration := parseDuration(card)
		stop := start.Add(duration)

		programs = append(programs, domain.Program{
			Channel:  channel.XmltvID,
			Source:   "programme-tv.net",
			Title:    title,
			SubTitle: subTitle,
			Category: category,
			Image:    image,
			Lang:     channel.Lang,
			Start:    start,
			Stop:     stop,
		})
	}

	return programs, nil
}

func parseStart(card *html.Node, date time.Time) time.Time {
	raw := textByClass(card, "mainBroadcastCard-startingHour")
	raw = strings.ReplaceAll(raw, "h", ":")
	dateStr := date.In(paris).Format("01/02/2006")
	s := dateStr + " " + raw
	t, _ := time.ParseInLocation("01/02/2006 15:04", s, paris)
	return t.UTC()
}

var durationRe = regexp.MustCompile(`(?:(\d+)h)?(\d+)min`)

func parseDuration(card *html.Node) time.Duration {
	raw := textByClass(card, "mainBroadcastCard-durationContent")
	m := durationRe.FindStringSubmatch(raw)
	if m == nil {
		return 0
	}
	var d time.Duration
	if m[1] != "" {
		h, _ := strconv.Atoi(m[1])
		d += time.Duration(h) * time.Hour
	}
	mins, _ := strconv.Atoi(m[2])
	d += time.Duration(mins) * time.Minute
	return d
}

func parseImage(card *html.Node) string {
	imgContent := findByClass(card, "mainBroadcastCard-imageContent")
	if imgContent == nil {
		return ""
	}

	img := findTag(imgContent, "img")
	if img == nil {
		return ""
	}

	srcset := attrVal(img, "srcset")
	if srcset == "" {
		srcset = attrVal(img, "data-srcset")
	}
	if srcset == "" {
		return ""
	}

	for _, source := range strings.Split(srcset, ",") {
		parts := strings.Fields(strings.TrimSpace(source))
		if len(parts) == 2 && parts[1] == "128w" {
			return strings.ReplaceAll(parts[0], "128x180", "960x540")
		}
	}

	return ""
}

func findAllByClass(n *html.Node, class string) []*html.Node {
	var results []*html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && hasClass(n, class) {
			results = append(results, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return results
}

func findByClass(n *html.Node, class string) *html.Node {
	var result *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if result != nil {
			return
		}
		if n.Type == html.ElementNode && hasClass(n, class) {
			result = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return result
}

func findTag(n *html.Node, tag string) *html.Node {
	var result *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if result != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == tag {
			result = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return result
}

func hasClass(n *html.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range strings.Fields(a.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

func textByClass(n *html.Node, class string) string {
	el := findByClass(n, class)
	if el == nil {
		return ""
	}
	return strings.TrimSpace(textContent(el))
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

func attrVal(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
