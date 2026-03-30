package jiotv

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

func (p *provider) Name() string        { return "jiotv.com" }
func (p *provider) Description() string { return "JioTV Indian streaming" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "jiotv.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	offset := int(date.UTC().Truncate(24*time.Hour).Sub(today).Hours() / 24)
	url := fmt.Sprintf("https://jiotvapi.cdn.jio.com/apis/v1.3/getepg/get?channel_id=%s&offset=%d", channel.SiteID, offset)

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
		return nil, fmt.Errorf("HTTP %d from jiotv", resp.StatusCode)
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

	programs := make([]domain.Program, 0, len(response.EPG))
	for _, item := range response.EPG {
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "jiotv.com",
			Title:   item.ShowName,
			Lang:    channel.Lang,
		}

		if item.EpisodeDesc != "" {
			p.Description = item.EpisodeDesc
		} else if item.Description != "" {
			p.Description = item.Description
		}

		if item.EpisodeNum > 0 {
			p.Episode = item.EpisodeNum
		}

		if item.EpisodePoster != "" {
			p.Image = fmt.Sprintf("https://jiotvimages.cdn.jio.com/dare_images/shows/700/-/%s", item.EpisodePoster)
		}

		if item.StartEpoch > 0 {
			p.Start = time.UnixMilli(item.StartEpoch).UTC()
		}
		if item.EndEpoch > 0 {
			p.Stop = time.UnixMilli(item.EndEpoch).UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	EPG []epgItem `json:"epg"`
}

type epgItem struct {
	ShowName      string `json:"showname"`
	Description   string `json:"description"`
	EpisodeDesc   string `json:"episode_desc"`
	EpisodeNum    int    `json:"episode_num"`
	EpisodePoster string `json:"episodePoster"`
	StartEpoch    int64  `json:"startEpoch"`
	EndEpoch      int64  `json:"endEpoch"`
}
