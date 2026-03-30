package tvlv

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
	plugin.Register(&provider{client: httpclient.Browser()})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "tv.lv" }
func (p *provider) Description() string { return "Latvian TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tv.lv")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()
	u := fmt.Sprintf("https://www.tv.lv/programme/listing/none/%s?filter=channel&subslug=%s",
		d.Format("02-01-2006"), channel.SiteID)

	var data []byte
	backoff := time.Second
	const maxRetries = 5

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}

		resp, err := p.client.Do(req)
		if err != nil {
			return nil, err
		}

		data, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			if attempt == maxRetries {
				return nil, nil
			}
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP %d from tvlv", resp.StatusCode)
		}

		break
	}

	return Parse(data, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil
	}

	programmes := response.Schedule.Programme
	programs := make([]domain.Program, 0, len(programmes))
	for _, item := range programmes {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tv.lv",
			Title:       item.Title,
			Description: item.DescriptionLong,
			Category:    item.CategoryString,
			Image:       item.Image,
			Lang:        channel.Lang,
		}
		if item.StartUnix != 0 {
			p.Start = time.Unix(item.StartUnix, 0).UTC()
		}
		if item.StopUnix != 0 {
			p.Stop = time.Unix(item.StopUnix, 0).UTC()
		}
		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Schedule schedule `json:"schedule"`
}

type schedule struct {
	Programme []programme `json:"programme"`
}

type programme struct {
	Title          string `json:"title"`
	DescriptionLong string `json:"description_long"`
	CategoryString string `json:"categorystring"`
	Image          string `json:"image"`
	StartUnix      int64  `json:"start_unix"`
	StopUnix       int64  `json:"stop_unix"`
}
