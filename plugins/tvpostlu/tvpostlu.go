package tvpostlu

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

func NewWithClient(client *http.Client) *provider {
	return &provider{client: client}
}

func (p *provider) Name() string        { return "tv.post.lu" }
func (p *provider) Description() string { return "POST Luxembourg TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tv.post.lu")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	day := date.Truncate(24 * time.Hour).UTC()

	url := fmt.Sprintf(
		"https://tv.post.lu/api/channels?id=%s&date=%s",
		channel.SiteID,
		day.Format("2006-01-02"),
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from tvpostlu", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	if response.Result == nil || response.Result.EPG == nil {
		return nil, nil
	}

	items := response.Result.EPG.Programme
	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tv.post.lu",
			Title:       item.Title,
			Description: item.Description,
			Category:    item.ProgramType,
			Image:       item.ImageURL,
			Lang:        channel.Lang,
			Start:       time.Unix(item.TsStart, 0).UTC(),
			Stop:        time.Unix(item.TsEnd, 0).UTC(),
		}
		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Result *apiResult `json:"result"`
}

type apiResult struct {
	EPG *apiEPG `json:"epg"`
}

type apiEPG struct {
	Programme []programme `json:"programme"`
}

type programme struct {
	TsStart     int64  `json:"tsStart"`
	TsEnd       int64  `json:"tsEnd"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ProgramType string `json:"program_type"`
	ImageURL    string `json:"image_url"`
}
