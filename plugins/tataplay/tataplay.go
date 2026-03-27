package tataplay

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
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

func (p *provider) Name() string        { return "tataplay.com" }
func (p *provider) Description() string { return "Tata Play India TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "tataplay-1001", Name: "Star Plus", Source: "tataplay.com", SiteID: "1001", Lang: "en", XmltvID: "StarPlus.in"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := "https://tm.tapi.videoready.tv/content-detail/pub/api/v2/channels/schedule?date=" + date.Format("02-01-2006")

	body := `{"id":"` + channel.SiteID + `"}`
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil
	}

	items := response.Data.EPG
	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		prog := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tataplay.com",
			Title:       item.Title,
			Description: item.Desc,
			Category:    item.Category,
			Image:       item.BoxCoverImage,
			Lang:        channel.Lang,
		}

		if item.StartTime != "" {
			if t, err := time.Parse(time.RFC3339Nano, item.StartTime); err == nil {
				prog.Start = t
			} else if t, err := time.Parse("2006-01-02T15:04:05.000Z", item.StartTime); err == nil {
				prog.Start = t
			}
		}

		if item.EndTime != "" {
			if t, err := time.Parse(time.RFC3339Nano, item.EndTime); err == nil {
				prog.Stop = t
			} else if t, err := time.Parse("2006-01-02T15:04:05.000Z", item.EndTime); err == nil {
				prog.Stop = t
			}
		}

		programs = append(programs, prog)
	}

	return programs, nil
}

type apiResponse struct {
	Data responseData `json:"data"`
}

type responseData struct {
	EPG []epgItem `json:"epg"`
}

type epgItem struct {
	Title         string `json:"title"`
	StartTime     string `json:"startTime"`
	EndTime       string `json:"endTime"`
	Desc          string `json:"desc"`
	Category      string `json:"category"`
	BoxCoverImage string `json:"boxCoverImage"`
}
