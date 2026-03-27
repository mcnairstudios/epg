package movistarpe

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

func (p *provider) Name() string        { return "tv.movistar.com.pe" }
func (p *provider) Description() string { return "Movistar TV Peru" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tv.movistar.com.pe")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://contentapi-pe.cdn.telefonica.com/28/default/es-PE/schedules?fields=Pid,Title,Description,ChannelName,LiveChannelPid,Start,End,images.videoFrame,AgeRatingPid&orderBy=START_TIME:a&filteravailability=false&starttime=%d&endtime=%d&livechannelpids=%s",
		date.Unix(), date.AddDate(0, 0, 1).Unix(), channel.SiteID,
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

	programs := make([]domain.Program, 0, len(response.Content))
	for _, item := range response.Content {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tv.movistar.com.pe",
			Title:       item.Title,
			Description: item.Description,
			Lang:        channel.Lang,
			Start:       time.Unix(item.Start, 0).UTC(),
			Stop:        time.Unix(item.End, 0).UTC(),
		}
		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Content []contentItem `json:"Content"`
}

type contentItem struct {
	Title       string `json:"Title"`
	Description string `json:"Description"`
	Start       int64  `json:"Start"`
	End         int64  `json:"End"`
}
