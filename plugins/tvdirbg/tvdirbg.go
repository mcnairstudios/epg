package tvdirbg

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

var sofiaTZ, _ = time.LoadLocation("Europe/Sofia")

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

func (p *provider) Name() string        { return "tv.dir.bg" }
func (p *provider) Description() string { return "Dir.bg Bulgarian TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "tvdirbg-61", Name: "bTV", Source: "tv.dir.bg", SiteID: "61", Lang: "bg", XmltvID: "BTV.bg"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	req, err := http.NewRequest(http.MethodPost, "https://tv.dir.bg/load/programs", nil)
	if err != nil {
		return nil, err
	}

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

var spaceRe = regexp.MustCompile(`\s+`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	var resp jsonResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return []domain.Program{}, nil
	}

	if !resp.Status {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(bytes.NewReader([]byte(resp.HTML)))
	if err != nil {
		return []domain.Program{}, nil
	}

	items := findNodesByClass(doc, "broadcast-item")
	d := date.Truncate(24 * time.Hour).UTC()
	programs := make([]domain.Program, 0)

	for _, item := range items {
		timeNodes := findNodesByClass(item, "broadcast-time")
		titleNodes := findNodesByClass(item, "broadcast-title")

		if len(timeNodes) == 0 || len(titleNodes) == 0 {
			continue
		}

		timeText := strings.TrimSpace(collectText(timeNodes[0]))
		title := spaceRe.ReplaceAllString(strings.TrimSpace(collectText(titleNodes[0])), " ")

		if timeText == "" {
			continue
		}

		start, err := time.ParseInLocation("2006-01-02 15:04", d.Format("2006-01-02")+" "+timeText, sofiaTZ)
		if err != nil {
			continue
		}

		if len(programs) > 0 {
			prev := &programs[len(programs)-1]
			if start.Before(prev.Start) {
				d = d.Add(24 * time.Hour)
				start, _ = time.ParseInLocation("2006-01-02 15:04", d.Format("2006-01-02")+" "+timeText, sofiaTZ)
			}
			prev.Stop = start.UTC()
		}

		stop := start.Add(30 * time.Minute)

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "tv.dir.bg",
			Title:   title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type jsonResponse struct {
	Status bool   `json:"status"`
	HTML   string `json:"html"`
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
