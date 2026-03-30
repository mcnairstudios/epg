package distrotv

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

func (p *provider) Name() string        { return "distro.tv" }
func (p *provider) Description() string { return "DistroTV free streaming" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "distro.tv")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	diff := int(date.Sub(now).Hours() / 24)

	rangeStr := "now,24h"
	if diff > 0 {
		rangeStr = fmt.Sprintf("%dh,%dh", diff*24, (diff+1)*24)
	}

	url := fmt.Sprintf("https://tv.jsrdn.com/epg/query.php?range=%s&id=%s,", rangeStr, channel.SiteID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36 Edg/144.0.0.0")
	req.Header.Set("Referer", "https://distro.tv/")
	req.Header.Set("Origin", "https://distro.tv")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from distrotv", resp.StatusCode)
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

	chData, ok := response.EPG[channel.SiteID]
	if !ok {
		return nil, nil
	}

	programs := make([]domain.Program, 0, len(chData.Slots))
	for _, slot := range chData.Slots {
		if slot.Title == "" {
			continue
		}

		start, err := time.Parse("2006-01-02 15:04:05", slot.Start)
		if err != nil {
			continue
		}
		stop, err := time.Parse("2006-01-02 15:04:05", slot.End)
		if err != nil {
			continue
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "distro.tv",
			Title:       slot.Title,
			Description: slot.Description,
			Image:       slot.ImgThumbH,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

type apiResponse struct {
	EPG map[string]channelData `json:"epg"`
}

type channelData struct {
	Title string     `json:"title"`
	Slots []slotData `json:"slots"`
}

type slotData struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ImgThumbH   string `json:"img_thumbh"`
	Start       string `json:"start"`
	End         string `json:"end"`
}
