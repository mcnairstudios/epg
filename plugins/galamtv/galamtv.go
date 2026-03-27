package galamtv

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

var almatyTZ *time.Location

func init() {
	var err error
	almatyTZ, err = time.LoadLocation("Asia/Almaty")
	if err != nil {
		almatyTZ = time.FixedZone("Asia/Almaty", 6*60*60)
	}
}

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

func (p *provider) Name() string        { return "galamtv.kz" }
func (p *provider) Description() string { return "Galam TV Kazakhstan IPTV service" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "galamtv.kz")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.In(almatyTZ)
	startOfDay := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, almatyTZ)
	endOfDay := startOfDay.Add(24 * time.Hour)

	url := fmt.Sprintf(
		"https://galam.server-api.lfstrm.tv/channels/%s/programs?period=%d:%d",
		channel.SiteID, startOfDay.Unix(), endOfDay.Unix(),
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", "https://galamtv.kz/")
	req.Header.Set("Origin", "https://galamtv.kz")
	req.Header.Set("Accept", "*/*")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
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

	programs := make([]domain.Program, 0, len(response.Programs))
	for _, item := range response.Programs {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "galamtv.kz",
			Title:       item.MetaInfo.Title,
			Description: item.MetaInfo.Description,
			Lang:        channel.Lang,
			Start:       time.Unix(item.ScheduleInfo.Start, 0).UTC(),
			Stop:        time.Unix(item.ScheduleInfo.End, 0).UTC(),
		}

		if len(item.MediaInfo.Thumbnails) > 0 {
			p.Image = item.MediaInfo.Thumbnails[0].URL
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Programs []programItem `json:"programs"`
}

type programItem struct {
	MetaInfo     metaInfo     `json:"metaInfo"`
	ScheduleInfo scheduleInfo `json:"scheduleInfo"`
	MediaInfo    mediaInfo    `json:"mediaInfo"`
}

type metaInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type scheduleInfo struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

type mediaInfo struct {
	Thumbnails []thumbnail `json:"thumbnails"`
}

type thumbnail struct {
	URL string `json:"url"`
}
