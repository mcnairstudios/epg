package rthkhk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

var hongKong *time.Location

func init() {
	var err error
	hongKong, err = time.LoadLocation("Asia/Hong_Kong")
	if err != nil {
		panic(err)
	}

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

func (p *provider) Name() string        { return "rthk.hk" }
func (p *provider) Description() string { return "RTHK Hong Kong public broadcaster" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "rthk.hk")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.In(hongKong).Format("20060102")
	url := fmt.Sprintf("https://www.rthk.hk/timetable/main_timetable/%s", d)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", fmt.Sprintf("lang=%s", channel.Lang))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from rthkhk", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

var (
	startTimeRe = regexp.MustCompile(`^(\d+):(\d+)-`)
	stopTimeRe  = regexp.MustCompile(`-(\d+):(\d+)$`)
	cateRe      = regexp.MustCompile(`^\|(.*)\|$`)
)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil
	}

	if response.Result == nil {
		return nil, nil
	}

	var channelHTML string
	for _, item := range response.Result {
		if fmt.Sprint(item.Key) == channel.SiteID {
			channelHTML = item.Data
			break
		}
	}

	if channelHTML == "" {
		return nil, nil
	}

	doc, err := html.Parse(strings.NewReader(channelHTML))
	if err != nil {
		return nil, nil
	}

	showWraps := findAllByClass(doc, "showWrap")
	programs := make([]domain.Program, 0, len(showWraps))
	dateInHK := date.In(hongKong)

	for _, wrap := range showWraps {
		title := attrBySelector(wrap, "showTit", "title")
		subTitle := attrBySelector(wrap, "showEpi", "title")
		image := dataAttr(wrap, "single-wrap", "p")
		categories := parseCategories(wrap)
		timeRow := textByClass(wrap, "timeRow")

		start := parseTimeFromRow(timeRow, dateInHK, startTimeRe)
		stop := parseTimeFromRow(timeRow, dateInHK, stopTimeRe)

		if start.IsZero() {
			continue
		}

		p := domain.Program{
			Channel:  channel.XmltvID,
			Source:   "rthk.hk",
			Title:    title,
			SubTitle: subTitle,
			Image:    image,
			Lang:     channel.Lang,
			Start:    start.UTC(),
			Stop:     stop.UTC(),
		}

		if len(categories) > 0 {
			p.Category = strings.Join(categories, ", ")
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func parseTimeFromRow(timeRow string, date time.Time, re *regexp.Regexp) time.Time {
	timeRow = strings.TrimSpace(timeRow)
	m := re.FindStringSubmatch(timeRow)
	if m == nil || len(m) < 3 {
		return time.Time{}
	}
	dateStr := date.Format("2006-01-02")
	t, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+m[1]+":"+m[2], hongKong)
	if err != nil {
		return time.Time{}
	}
	return t
}

func parseCategories(wrap *html.Node) []string {
	cateStr := dataAttr(wrap, "single-wrap", "cate")
	if cateStr == "" {
		return nil
	}
	m := cateRe.FindStringSubmatch(cateStr)
	if m == nil || len(m) < 2 {
		return nil
	}
	parts := strings.Split(m[1], "||")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func dataAttr(n *html.Node, class, attr string) string {
	el := findByClass(n, class)
	if el == nil {
		return ""
	}
	return attrVal(el, "data-"+attr)
}

func attrBySelector(n *html.Node, class, attr string) string {
	el := findByClass(n, class)
	if el == nil {
		return ""
	}
	return attrVal(el, attr)
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

type apiResponse struct {
	Result []resultItem `json:"result"`
}

type resultItem struct {
	Key  interface{} `json:"key"`
	Data string      `json:"data"`
}
