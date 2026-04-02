package tvpassport

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

func (p *provider) Name() string        { return "tvpassport.com" }
func (p *provider) Description() string { return "TV Passport North American TV listings" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvpassport.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://www.tvpassport.com/tv-listings/stations/%s/%s",
		channel.SiteID,
		date.Format("2006-01-02"),
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
		return nil, fmt.Errorf("HTTP %d from tvpassport", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	tz := findTimezone(doc)
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, _ = time.LoadLocation("America/New_York")
	}

	items := findListGroupItems(doc)
	programs := make([]domain.Program, 0, len(items))

	for _, item := range items {
		attrs := collectDataAttrs(item)

		title := attrs["showname"]
		subtitle := attrs["episodetitle"]
		if title == "Movie" || title == "Cinéma" {
			title = subtitle
			subtitle = ""
		}

		startStr := attrs["st"]
		durationStr := attrs["duration"]

		start, _ := time.ParseInLocation("2006-01-02 15:04:05", startStr, loc)
		duration, _ := strconv.Atoi(durationStr)
		stop := start.Add(time.Duration(duration) * time.Minute)

		image := ""
		if pic := attrs["showpicture"]; pic != "" {
			image = "https://cdn.tvpassport.com/image/show/960x540/" + pic
		}

		category := attrs["showtype"]

		rating := attrs["rating"]
		ratingVal := ""
		if rating != "" {
			re := regexp.MustCompile(`^TV`)
			ratingVal = re.ReplaceAllString(rating, "TV-")
		}

		year := attrs["year"]

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tvpassport.com",
			Title:       title,
			SubTitle:    subtitle,
			Description: attrs["description"],
			Image:       image,
			Category:    category,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		}

		_ = ratingVal
		_ = year
		_ = attrs["cast"]
		_ = attrs["director"]
		_ = attrs["guest"]

		programs = append(programs, p)
	}

	return programs, nil
}

func findTimezone(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "select" {
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == "timezone_selector" {
				return findSelectedOption(n)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if tz := findTimezone(c); tz != "" {
			return tz
		}
	}
	return ""
}

func findSelectedOption(sel *html.Node) string {
	for c := sel.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "option" {
			selected := false
			val := ""
			for _, a := range c.Attr {
				if a.Key == "selected" {
					selected = true
				}
				if a.Key == "value" {
					val = a.Val
				}
			}
			if selected {
				return val
			}
		}
	}
	return "America/New_York"
}

func findListGroupItems(n *html.Node) []*html.Node {
	var items []*html.Node
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if a.Key == "class" && strings.Contains(a.Val, "list-group-item") {
				items = append(items, n)
				return items
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		items = append(items, findListGroupItems(c)...)
	}
	return items
}

func collectDataAttrs(n *html.Node) map[string]string {
	attrs := make(map[string]string)
	for _, a := range n.Attr {
		if strings.HasPrefix(a.Key, "data-") {
			key := strings.TrimPrefix(a.Key, "data-")
			key = strings.ToLower(strings.ReplaceAll(key, "-", ""))
			attrs[key] = a.Val
		}
	}
	return attrs
}

func (p *provider) Timezone() string { return "America/New_York" }
