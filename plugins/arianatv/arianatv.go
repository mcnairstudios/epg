package arianatv

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

var kabulTZ = time.FixedZone("Asia/Kabul", 4*3600+30*60)

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "arianatelevision.com" }
func (p *provider) Description() string { return "Ariana Television program schedule" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "arianatelevision.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	req, err := http.NewRequest(http.MethodGet, "https://www.arianatelevision.com/program-schedule/", nil)
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

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	textarea := findNodeByID(doc, "jtrt_table_settings_508")
	if textarea == nil {
		return []domain.Program{}, nil
	}

	content := textContent(textarea)
	if strings.TrimSpace(content) == "" {
		return []domain.Program{}, nil
	}

	var tableData [][][]string
	if err := json.Unmarshal([]byte(content), &tableData); err != nil {
		return []domain.Program{}, nil
	}

	if len(tableData) == 0 || len(tableData[0]) == 0 {
		return []domain.Program{}, nil
	}

	rows := tableData[0]
	if len(rows) < 2 {
		return []domain.Program{}, nil
	}

	dayCol := int(date.Weekday()) + 2
	if dayCol > 7 {
		dayCol = 1
	}

	rows = rows[1:]

	type item struct {
		timeStr string
		title   string
	}

	var items []item
	for _, row := range rows {
		if len(row) <= dayCol {
			continue
		}
		t := strings.TrimSpace(row[0])
		title := strings.TrimSpace(row[dayCol])
		if t == "" || title == "" {
			continue
		}
		items = append(items, item{timeStr: t, title: title})
	}

	dateStr := date.Format("2006-01-02")
	currentDate := date

	programs := make([]domain.Program, 0, len(items))
	for i, it := range items {
		start, err := time.ParseInLocation("2006-01-02 15:04", fmt.Sprintf("%s %s", dateStr, it.timeStr), kabulTZ)
		if err != nil {
			start, err = time.ParseInLocation("2006-01-02 15:4", fmt.Sprintf("%s %s", dateStr, it.timeStr), kabulTZ)
			if err != nil {
				continue
			}
		}

		if len(programs) > 0 && start.Before(programs[len(programs)-1].Start) {
			currentDate = currentDate.AddDate(0, 0, 1)
			dateStr = currentDate.Format("2006-01-02")
			start, err = time.ParseInLocation("2006-01-02 15:04", fmt.Sprintf("%s %s", dateStr, it.timeStr), kabulTZ)
			if err != nil {
				start, _ = time.ParseInLocation("2006-01-02 15:4", fmt.Sprintf("%s %s", dateStr, it.timeStr), kabulTZ)
			}
		}

		if len(programs) > 0 {
			programs[len(programs)-1].Stop = start.UTC()
		}

		stop := start.Add(30 * time.Minute)

		_ = i
		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "arianatelevision.com",
			Title:   it.title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		})
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
		if found := findNodeByID(c, id); found != nil {
			return found
		}
	}
	return nil
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
