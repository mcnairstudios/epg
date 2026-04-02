package siolnet

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

func (p *provider) Name() string        { return "tv-spored.siol.net" }
func (p *provider) Description() string { return "Siol TV Slovenia (312 channels)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tv-spored.siol.net")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://tv-spored.siol.net/kanal/%s/datum/%s", channel.SiteID, date.Format("20060102"))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from siolnet", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	events := extractEvents(data, date)
	if len(events) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(events))
	for _, e := range events {
		p := domain.Program{
			Channel:  channel.XmltvID,
			Source:   "tv-spored.siol.net",
			Title:    e.Title,
			Category: e.Category,
			Lang:     channel.Lang,
		}

		p.Season = intOrZero(e.Season)
		p.Episode = intOrZero(e.Episode)

		if e.StartDateTime != "" {
			if t, err := time.Parse(time.RFC3339Nano, e.StartDateTime); err == nil {
				p.Start = t.UTC()
			} else if t, err := time.Parse("2006-01-02T15:04:05.000Z", e.StartDateTime); err == nil {
				p.Start = t.UTC()
			}
		}

		if e.StopDateTime != "" {
			if t, err := time.Parse(time.RFC3339Nano, e.StopDateTime); err == nil {
				p.Stop = t.UTC()
			} else if t, err := time.Parse("2006-01-02T15:04:05.000Z", e.StopDateTime); err == nil {
				p.Stop = t.UTC()
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func intOrZero(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func extractEvents(data []byte, date time.Time) []eventEntry {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil
	}

	scripts := findAllScriptTexts(doc)

	for _, script := range scripts {
		if !strings.Contains(script, "channelsAsJson") {
			continue
		}

		dataStr := extractNextFData(script)
		if dataStr == "" {
			continue
		}

		channelEvents := findChannelsAsJson(dataStr)
		if channelEvents == nil || len(channelEvents) == 0 {
			continue
		}

		if channelEvents[0].Events == nil {
			continue
		}

		dateStr := date.Format("2006-01-02")
		var filtered []eventEntry
		for _, e := range channelEvents[0].Events {
			if strings.HasPrefix(e.StartDateTime, dateStr) {
				filtered = append(filtered, e)
			}
		}
		return filtered
	}

	return nil
}

func extractNextFData(script string) string {
	chunks := strings.Split(script, "self.__next_f.push(")

	for _, chunk := range chunks {
		if !strings.Contains(chunk, "channelsAsJson") {
			continue
		}

		str := extractJSString(chunk)
		if str == "" {
			continue
		}

		colonIdx := strings.Index(str, ":")
		if colonIdx < 0 {
			continue
		}

		return str[colonIdx+1:]
	}

	return ""
}

func extractJSString(chunk string) string {
	openBracket := strings.Index(chunk, "[")
	if openBracket < 0 {
		return ""
	}

	afterBracket := chunk[openBracket:]

	quoteIdx := -1
	for i := 0; i < len(afterBracket); i++ {
		if afterBracket[i] == '\'' {
			quoteIdx = i
			break
		}
	}
	if quoteIdx < 0 {
		return ""
	}

	var sb strings.Builder
	i := quoteIdx + 1
	for i < len(afterBracket) {
		ch := afterBracket[i]
		if ch == '\\' && i+1 < len(afterBracket) {
			next := afterBracket[i+1]
			switch next {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case '\\':
				sb.WriteByte('\\')
			case '\'':
				sb.WriteByte('\'')
			case '"':
				sb.WriteByte('"')
			default:
				sb.WriteByte(next)
			}
			i += 2
			continue
		}
		if ch == '\'' {
			break
		}
		sb.WriteByte(ch)
		i++
	}

	return sb.String()
}

func findAllScriptTexts(doc *html.Node) []string {
	var results []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			text := extractText(n)
			if text != "" {
				results = append(results, text)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return results
}

func findChannelsAsJson(jsonStr string) []channelEntry {
	var raw interface{}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil
	}

	result := searchForKey(raw, "channelsAsJson")
	if result == nil {
		return nil
	}

	b, err := json.Marshal(result)
	if err != nil {
		return nil
	}

	var entries []channelEntry
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil
	}
	return entries
}

func searchForKey(v interface{}, key string) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		if found, ok := val[key]; ok {
			return found
		}
		for _, child := range val {
			if result := searchForKey(child, key); result != nil {
				return result
			}
		}
	case []interface{}:
		for _, item := range val {
			if result := searchForKey(item, key); result != nil {
				return result
			}
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

type channelEntry struct {
	ExternalID string       `json:"externalId"`
	Name       string       `json:"name"`
	Events     []eventEntry `json:"events"`
}

type eventEntry struct {
	ExternalID    string   `json:"externalId"`
	Title         string   `json:"title"`
	Season        *int     `json:"season"`
	Episode       *int     `json:"episode"`
	Category      string   `json:"category"`
	SubCategories []string `json:"subCategories"`
	StartDateTime string   `json:"startDateTime"`
	StopDateTime  string   `json:"stopDateTime"`
	Duration      int      `json:"duration"`
	Date          string   `json:"date"`
}

func (p *provider) Timezone() string { return "Europe/Ljubljana" }
