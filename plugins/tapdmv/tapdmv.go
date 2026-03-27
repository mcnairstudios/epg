package tapdmv

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

func (p *provider) Name() string        { return "tapdmv.com" }
func (p *provider) Description() string { return "TapDMV streaming guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "tapdmv-1", Name: "TapDMV", Source: "tapdmv.com", SiteID: "1", Lang: "en", XmltvID: "TapDMV.us"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://epg.tapdmv.com/calendar/%s?%%24limit=10000&%%24sort%%5BcreatedAt%%5D=-1&start=%s&end=%s",
		channel.SiteID, date.Format(time.RFC3339), date.AddDate(0, 0, 1).Format(time.RFC3339),
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
	var items []programItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tapdmv.com",
			Title:       item.Program,
			Description: item.Description,
			Category:    item.Genre,
			Image:       item.ThumbnailImage,
			Lang:        channel.Lang,
		}

		if t, err := time.Parse(time.RFC3339Nano, item.StartTime); err == nil {
			p.Start = t.UTC()
		}
		if t, err := time.Parse(time.RFC3339Nano, item.EndTime); err == nil {
			p.Stop = t.UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type programItem struct {
	Program        string `json:"program"`
	Description    string `json:"description"`
	Genre          string `json:"genre"`
	ThumbnailImage string `json:"thumbnailImage"`
	StartTime      string `json:"startTime"`
	EndTime        string `json:"endTime"`
}
