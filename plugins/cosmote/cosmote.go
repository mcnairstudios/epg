package cosmote

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "cosmotetv.gr" }
func (p *provider) Description() string { return "Cosmote TV Greece" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "cosmote-vouli", Name: "Hellenic Parliament TV", Source: "cosmotetv.gr", SiteID: "vouli", Lang: "el", XmltvID: "HellenicParliamentTV.gr"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24*time.Hour - time.Second)
	url := fmt.Sprintf(
		"https://mwapi-prod.cosmotetvott.gr/api/v3.4/epg/listings/el?from=%d&to=%d&callSigns=%s&endingIncludedInRange=false",
		startOfDay.Unix(), endOfDay.Unix(), channel.SiteID,
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
	for _, ch := range response.Channels {
		for _, item := range ch.Items {
			start, err := time.Parse(time.RFC3339, item.StartTime)
			if err != nil {
				continue
			}
			stop, err := time.Parse(time.RFC3339, item.EndTime)
			if err != nil {
				continue
			}

			desc := item.Description
			if desc == "" {
				desc = "No description available"
			}

			programs = append(programs, domain.Program{
				Channel:     channel.XmltvID,
				Source:      "cosmotetv.gr",
				Title:       item.Title,
				Description: desc,
				Category:    item.Qoe.Genre,
				Image:       item.Thumbnails.Standard,
				Lang:        channel.Lang,
				Start:       start.UTC(),
				Stop:        stop.UTC(),
			})
		}
	}

	if programs == nil {
		programs = []domain.Program{}
	}

	return programs, nil
}

type apiResponse struct {
	Channels []channelData `json:"channels"`
}

type channelData struct {
	Items []programItem `json:"items"`
}

type programItem struct {
	StartTime   string     `json:"startTime"`
	EndTime     string     `json:"endTime"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Qoe         qoeData    `json:"qoe"`
	Thumbnails  thumbData  `json:"thumbnails"`
}

type qoeData struct {
	Genre string `json:"genre"`
}

type thumbData struct {
	Standard string `json:"standard"`
}
