package porthu

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

func (p *provider) Name() string        { return "port.hu" }
func (p *provider) Description() string { return "Port.hu Hungarian TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "port.hu")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Format("2006-01-02")
	url := fmt.Sprintf(
		"https://port.hu/tvapi?channel_id[]=tvchannel-%s&i_datetime_from=%s&i_datetime_to=%s",
		channel.SiteID, d, d,
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
		return nil, fmt.Errorf("HTTP %d from porthu", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return []domain.Program{}, nil
	}

	for _, v := range raw {
		var block dateBlock
		if err := json.Unmarshal(v, &block); err != nil {
			continue
		}
		for _, ch := range block.Channels {
			if ch.ID != fmt.Sprintf("tvchannel-%s", channel.SiteID) {
				continue
			}
			programs := make([]domain.Program, 0, len(ch.Programs))
			for _, item := range ch.Programs {
				p := domain.Program{
					Channel: channel.XmltvID,
					Source:  "port.hu",
					Title:   item.Title,
					Lang:    channel.Lang,
				}

				if item.EpisodeTitle != "" {
					p.SubTitle = item.EpisodeTitle
				}

				if item.Description != "" {
					p.Description = item.Description
				} else if item.ShortDescription != "" {
					p.Description = item.ShortDescription
				}

				if item.Restriction != nil && item.Restriction.Category != "" {
					p.Category = item.Restriction.Category
				}

				if item.StartTS != 0 {
					p.Start = time.Unix(int64(item.StartTS), 0).UTC()
				}

				if item.EndDatetime != "" {
					if t, err := time.Parse(time.RFC3339, item.EndDatetime); err == nil {
						p.Stop = t.UTC()
					}
				}

				programs = append(programs, p)
			}
			return programs, nil
		}
		break
	}

	return []domain.Program{}, nil
}

type dateBlock struct {
	Channels []channelBlock `json:"channels"`
}

type channelBlock struct {
	ID       string        `json:"id"`
	Programs []programItem `json:"programs"`
}

type programItem struct {
	Title            string       `json:"title"`
	EpisodeTitle     string       `json:"episode_title"`
	Description      string       `json:"description"`
	ShortDescription string       `json:"short_description"`
	StartTS          int64        `json:"start_ts"`
	EndDatetime      string       `json:"end_datetime"`
	Restriction      *restriction `json:"restriction"`
}

type restriction struct {
	Category string `json:"category"`
}
