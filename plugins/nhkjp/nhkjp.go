package nhkjp

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

func (p *provider) Name() string        { return "www3.nhk.or.jp" }
func (p *provider) Description() string { return "NHK World Japan EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "www3.nhk.or.jp")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Format("20060102")
	url := fmt.Sprintf("https://masterpl.hls.nhkworld.jp/epg/w/%s.json", d)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36")

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
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(resp.Data))
	for _, item := range resp.Data {
		start, err := time.Parse(time.RFC3339, item.StartTime)
		if err != nil {
			continue
		}
		stop, err := time.Parse(time.RFC3339, item.EndTime)
		if err != nil {
			continue
		}

		image := item.EpisodeThumbnailURL
		if image == "" {
			image = item.Thumbnail
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "www3.nhk.or.jp",
			Title:       item.Title,
			SubTitle:    item.EpisodeTitle,
			Description: item.Description,
			Image:       image,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		})
	}

	return programs, nil
}

type apiResponse struct {
	Data []scheduleItem `json:"data"`
}

type scheduleItem struct {
	Title               string `json:"title"`
	EpisodeTitle        string `json:"episodeTitle"`
	Description         string `json:"description"`
	Thumbnail           string `json:"thumbnail"`
	EpisodeThumbnailURL string `json:"episodeThumbnailURL"`
	StartTime           string `json:"startTime"`
	EndTime             string `json:"endTime"`
}
