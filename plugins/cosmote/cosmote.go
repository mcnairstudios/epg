package cosmote

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
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "cosmotetv.gr" }
func (p *provider) Description() string { return "Cosmote TV Greece TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "cosmotetv.gr")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	startOfDay := date.Truncate(24 * time.Hour).UTC().Unix()
	endOfDay := date.Truncate(24*time.Hour).Add(24*time.Hour-time.Second).UTC().Unix()

	url := fmt.Sprintf(
		"https://mwapi-prod.cosmotetvott.gr/api/v3.4/epg/listings/el?from=%d&to=%d&callSigns=%s&endingIncludedInRange=false",
		startOfDay, endOfDay, channel.SiteID,
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
			prog := domain.Program{
				Channel:     channel.XmltvID,
				Source:      "cosmotetv.gr",
				Title:       item.Title,
				Description: item.Description,
				Image:       item.Thumbnails.Standard,
				Lang:        channel.Lang,
			}

			if item.QOE.Genre != "" {
				prog.Category = item.QOE.Genre
			}

			if item.StartTime != "" {
				if t, err := time.Parse(time.RFC3339, item.StartTime); err == nil {
					prog.Start = t.UTC()
				}
			}
			if item.EndTime != "" {
				if t, err := time.Parse(time.RFC3339, item.EndTime); err == nil {
					prog.Stop = t.UTC()
				}
			}

			programs = append(programs, prog)
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
	QOE         qoeData    `json:"qoe"`
	Thumbnails  thumbnails `json:"thumbnails"`
}

type qoeData struct {
	Genre string `json:"genre"`
}

type thumbnails struct {
	Standard string `json:"standard"`
}
