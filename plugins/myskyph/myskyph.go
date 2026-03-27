package myskyph

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

var manilaTZ = time.FixedZone("Asia/Manila", 8*60*60)

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

func (p *provider) Name() string        { return "mysky.com.ph" }
func (p *provider) Description() string { return "MySky Philippines" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "mysky.com.ph")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	req, err := http.NewRequest(http.MethodGet, "https://skyepg.mysky.com.ph/Main/getEventsbyType", nil)
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
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	programs := make([]domain.Program, 0)
	for _, item := range response.Events {
		if item.Location != channel.SiteID {
			continue
		}

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "mysky.com.ph",
			Title:   item.Name,
			Lang:    channel.Lang,
		}

		if item.UserData.Description != "" {
			p.Description = item.UserData.Description
		}

		if t, err := time.ParseInLocation("2006/01/02 15:04", item.Start, manilaTZ); err == nil {
			p.Start = t.UTC()
		}
		if t, err := time.ParseInLocation("2006/01/02 15:04", item.End, manilaTZ); err == nil {
			p.Stop = t.UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Events []event `json:"events"`
}

type event struct {
	Name     string   `json:"name"`
	Location string   `json:"location"`
	Start    string   `json:"start"`
	End      string   `json:"end"`
	UserData userData `json:"userData"`
}

type userData struct {
	Description string `json:"description"`
}
