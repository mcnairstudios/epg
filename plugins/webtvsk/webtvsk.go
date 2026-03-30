package webtvsk

import (
	"bytes"
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

func (p *provider) Name() string        { return "webtv.sk" }
func (p *provider) Description() string { return "WebTV Slovakia (186 channels)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "webtv.sk")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	body, err := json.Marshal(map[string]interface{}{
		"channel_id": channel.SiteID,
		"date":       date.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.webtv.sk/epg/channel", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from webtvsk", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return []domain.Program{}, nil
	}

	if response.Content == nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(response.Content))
	for _, item := range response.Content {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "webtv.sk",
			Title:       item.Title,
			SubTitle:    item.Subtitle,
			Description: item.Description,
			Lang:        channel.Lang,
		}

		if len(item.Genres) > 0 {
			p.Category = item.Genres[0]
		}

		if t, err := time.Parse(time.RFC3339, item.Start); err == nil {
			p.Start = t.UTC()
		}
		if t, err := time.Parse(time.RFC3339, item.Stop); err == nil {
			p.Stop = t.UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	ID      string    `json:"id"`
	Content []epgItem `json:"content"`
}

type epgItem struct {
	Title       string   `json:"Title"`
	Subtitle    string   `json:"Subtitle"`
	Description string   `json:"Description"`
	Start       string   `json:"Start"`
	Stop        string   `json:"Stop"`
	Genres      []string `json:"Genres"`
}
