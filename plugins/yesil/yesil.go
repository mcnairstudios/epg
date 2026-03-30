package yesil

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

func (p *provider) Name() string        { return "yes.co.il" }
func (p *provider) Description() string { return "YES Israeli satellite TV provider" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "yes.co.il")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://svc.yes.co.il/api/content/broadcast-schedule/channels/%s?date=%d-%d-%d&ignorePastItems=true",
		channel.SiteID,
		date.Year(), int(date.Month()), date.Day(),
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Linux x86_64) AppleWebKit/600.3 (KHTML, like Gecko) Chrome/48.0.2544.291 Safari/600")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from yesil", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	var programs []domain.Program
	for _, item := range response.Items {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "yes.co.il",
			Title:       item.Title,
			Description: item.Description,
			Image:       item.ImageURL,
			Lang:        channel.Lang,
		}

		if item.Starts != "" {
			if t, err := time.Parse(time.RFC3339, item.Starts); err == nil {
				p.Start = t
			}
		}
		if item.Ends != "" {
			if t, err := time.Parse(time.RFC3339, item.Ends); err == nil {
				p.Stop = t
			}
		}

		programs = append(programs, p)
	}

	if programs == nil {
		programs = []domain.Program{}
	}

	return programs, nil
}

type apiResponse struct {
	Items []item `json:"items"`
}

type item struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"imageUrl"`
	Starts      string `json:"starts"`
	Ends        string `json:"ends"`
	ChannelID   string `json:"channelId"`
}
