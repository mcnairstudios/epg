package ctcru

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
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

func (p *provider) Name() string        { return "ctc.ru" }
func (p *provider) Description() string { return "CTC Russian TV channel" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "ctc.ru")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Format("02-01-2006")
	url := fmt.Sprintf("https://ctc.ru/api/page/v2/programm/?date=%s", d)

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
		return nil, fmt.Errorf("HTTP %d from ctcru", resp.StatusCode)
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

	var page apiPage
	if err := json.Unmarshal(data, &page); err != nil {
		return []domain.Program{}, nil
	}

	var node *contentNode
	for i := range page.Content {
		if page.Content[i].Type == "tv-program" {
			node = &page.Content[i]
			break
		}
	}
	if node == nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(node.Widgets))
	for _, w := range node.Widgets {
		start, err := time.Parse(time.RFC3339, w.StartTime)
		if err != nil {
			continue
		}
		stop, err := time.Parse(time.RFC3339, w.EndTime)
		if err != nil {
			continue
		}

		p := domain.Program{
			Channel:  channel.XmltvID,
			Source:   "ctc.ru",
			Title:    w.BubbleTitle,
			Category: w.BubbleSubTitle,
			Lang:     channel.Lang,
			Start:    start,
			Stop:     stop,
		}

		if w.BubbleURL != "" {
			p.Description = "https://ctc.ru" + w.BubbleURL
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiPage struct {
	Content []contentNode `json:"content"`
}

type contentNode struct {
	Type    string   `json:"type"`
	Widgets []widget `json:"widgets"`
}

type widget struct {
	BubbleTitle    string `json:"bubbleTitle"`
	BubbleSubTitle string `json:"bubbleSubTitle"`
	BubbleURL      string `json:"bubbleUrl"`
	StartTime      string `json:"startTime"`
	EndTime        string `json:"endTime"`
	IsPremiere     bool   `json:"isPremiere"`
	AgeLimit       *int   `json:"ageLimit"`
}

func (p *provider) Timezone() string { return "Europe/Moscow" }
