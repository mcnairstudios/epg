package nowplayer

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

func (p *provider) Name() string        { return "nowplayer.now.com" }
func (p *provider) Description() string { return "Now Player Hong Kong" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "nowplayer.now.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	diff := int(date.Truncate(24*time.Hour).Sub(time.Now().UTC().Truncate(24*time.Hour)).Hours()/24) + 1

	url := fmt.Sprintf("https://nowplayer.now.com/tvguide/epglist?channelIdList[]=%s&day=%d", channel.SiteID, diff)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", fmt.Sprintf("LANG=%s; Expires=null; Path=/; Domain=nowplayer.now.com", channel.Lang))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from nowplayer", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var outer []json.RawMessage
	if err := json.Unmarshal(data, &outer); err != nil {
		return []domain.Program{}, nil
	}

	if len(outer) == 0 {
		return []domain.Program{}, nil
	}

	var items []epgItem
	if err := json.Unmarshal(outer[0], &items); err != nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "nowplayer.now.com",
			Title:   item.Name,
			Lang:    channel.Lang,
		}

		if item.Start != 0 {
			p.Start = time.UnixMilli(item.Start).UTC()
		}
		if item.End != 0 {
			p.Stop = time.UnixMilli(item.End).UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type epgItem struct {
	Name  string `json:"name"`
	Start int64  `json:"start"`
	End   int64  `json:"end"`
}

func (p *provider) Timezone() string { return "Asia/Hong_Kong" }
