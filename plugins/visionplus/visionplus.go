package visionplus

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

func (p *provider) Name() string        { return "visionplus.id" }
func (p *provider) Description() string { return "Vision+ Indonesian streaming" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "visionplus-1", Name: "Vision+", Source: "visionplus.id", SiteID: "00000000000000000079", Lang: "id", XmltvID: "VisionPlus.id"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	lang := "IND"
	if channel.Lang == "en" {
		lang = "ENG"
	}

	url := fmt.Sprintf(
		"https://www.visionplus.id/managetv/tvinfo/events/schedule?language=%s&serviceId=%s&start=%sT00%%3A00%%3A00Z&end=%sT00%%3A00%%3A00Z&view=cd-events-grid-view",
		lang,
		channel.SiteID,
		date.Format("2006-01-02"),
		date.AddDate(0, 0, 1).Format("2006-01-02"),
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

	programs := make([]domain.Program, 0, len(response.Events))
	for _, ev := range response.Events {
		if ev.SID != channel.SiteID {
			continue
		}

		title := ev.Content.OriginalTitle
		if ev.Content.Loc != nil && len(ev.Content.Loc) > 0 && ev.Content.Loc[0].Title != "" {
			title = ev.Content.Loc[0].Title
		}

		var description string
		if ev.Content.Loc != nil && len(ev.Content.Loc) > 0 {
			description = ev.Content.Loc[0].Synopsis
		}

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "visionplus.id",
			Title:       title,
			Description: description,
			Lang:        channel.Lang,
		}

		if t, err := time.Parse(time.RFC3339, ev.Start); err == nil {
			p.Start = t.UTC()
		}
		if t, err := time.Parse(time.RFC3339, ev.End); err == nil {
			p.Stop = t.UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Events []event `json:"evs"`
}

type event struct {
	SID     string  `json:"sid"`
	Start   string  `json:"sta"`
	End     string  `json:"end"`
	Content content `json:"con"`
}

type content struct {
	OriginalTitle string       `json:"oti"`
	Categories    []string     `json:"categories"`
	Loc           []locContent `json:"loc"`
}

type locContent struct {
	Title    string `json:"tit"`
	Synopsis string `json:"syn"`
}
