package tvtvus

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
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

func (p *provider) Name() string        { return "tvtv.us" }
func (p *provider) Description() string { return "TVTV US TV listings" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvtv.us")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour).UTC()
	end := start.Add(24 * time.Hour)

	url := fmt.Sprintf(
		"https://www.tvtv.us/api/v1/lineup/USA-NY71652-X/grid/%s/%s/%s",
		start.Format(time.RFC3339Nano),
		end.Format(time.RFC3339Nano),
		channel.SiteID,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

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

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var outer []json.RawMessage
	if err := json.Unmarshal(data, &outer); err != nil {
		return []domain.Program{}, nil
	}

	if len(outer) == 0 {
		return []domain.Program{}, nil
	}

	var items []gridItem
	if err := json.Unmarshal(outer[0], &items); err != nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		start, err := time.Parse(time.RFC3339, item.StartTime)
		if err != nil {
			start, err = time.Parse("2006-01-02T15:04Z", item.StartTime)
			if err != nil {
				continue
			}
		}
		stop := start.Add(time.Duration(item.Duration) * time.Minute)

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "tvtv.us",
			Title:   item.Title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		}

		if item.Subtitle != "" {
			p.SubTitle = item.Subtitle
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type gridItem struct {
	StartTime string `json:"startTime"`
	Duration  int    `json:"duration"`
	Title     string `json:"title"`
	Subtitle  string `json:"subtitle"`
	ProgramID string `json:"programId"`
}
