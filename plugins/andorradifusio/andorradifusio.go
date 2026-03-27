package andorradifusio

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

//go:embed channels.json
var channelsData []byte

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

func (p *provider) Name() string        { return "andorradifusio.ad" }
func (p *provider) Description() string { return "Andorra Difusio TV guide (RTVA)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "andorradifusio.ad")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://www.andorradifusio.ad/programacio/%s", channel.SiteID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
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

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	jsonData := extractNextData(data)
	if jsonData == "" {
		return []domain.Program{}, nil
	}

	var page nextDataWrapper
	if err := json.Unmarshal([]byte(jsonData), &page); err != nil {
		return []domain.Program{}, nil
	}

	shortTitle := strings.ToUpper(channel.SiteID)

	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.AddDate(0, 0, 1)

	var programs []domain.Program
	for _, body := range page.Props.PageProps.Page.Body {
		if body.Name != "ShowScheduleRepresentation" {
			continue
		}
		for _, item := range body.Body {
			if item.ContentShortTitle != shortTitle {
				continue
			}
			start := parseDateTime(item.StartDateTime)
			stop := parseDateTime(item.EndDateTime)
			if start.IsZero() || stop.IsZero() {
				continue
			}
			if start.Before(dayStart) || !start.Before(dayEnd) {
				continue
			}
			title := strings.TrimSpace(item.Title)
			if title == "" {
				continue
			}
			programs = append(programs, domain.Program{
				Channel: channel.XmltvID,
				Source:  "andorradifusio.ad",
				Title:   title,
				Image:   item.MainImage,
				Lang:    channel.Lang,
				Start:   start,
				Stop:    stop,
			})
		}
	}

	if programs == nil {
		return []domain.Program{}, nil
	}
	return programs, nil
}

func parseDateTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	s = strings.TrimSuffix(s, " UTC")
	t, err := time.Parse("2006-01-02 15:04:05 -0700", s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func extractNextData(data []byte) string {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return ""
	}

	var result string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if result != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "script" {
			for _, a := range n.Attr {
				if a.Key == "id" && a.Val == "__NEXT_DATA__" {
					if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
						result = n.FirstChild.Data
					}
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return result
}

type nextDataWrapper struct {
	Props struct {
		PageProps struct {
			Page struct {
				Body []pageBody `json:"body"`
			} `json:"page"`
		} `json:"pageProps"`
	} `json:"props"`
}

type pageBody struct {
	Name string         `json:"name"`
	Body []scheduleItem `json:"body"`
}

type scheduleItem struct {
	ContentShortTitle string `json:"content_shorttitle"`
	StartDateTime     string `json:"start_date_time"`
	EndDateTime       string `json:"end_date_time"`
	Title             string `json:"title"`
	SubTitle          string `json:"subtitle"`
	MainImage         string `json:"main_image"`
}
