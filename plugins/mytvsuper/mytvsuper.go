package mytvsuper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var hkLocation *time.Location

func init() {
	hkLocation, _ = time.LoadLocation("Asia/Hong_Kong")
}

func (p *provider) Name() string        { return "mytvsuper.com" }
func (p *provider) Description() string { return "myTV SUPER Hong Kong EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "mytvsuper-j2", Name: "J2", Source: "mytvsuper.com", SiteID: "B", Lang: "zh", XmltvID: "J2.hk"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	dateStr := date.Format("20060102")
	url := fmt.Sprintf(
		"https://content-api.mytvsuper.com/v1/epg?network_code=%s&from=%s&to=%s&platform=web",
		channel.SiteID, dateStr, dateStr,
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

	return Parse(body, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	var networks []networkData
	if err := json.Unmarshal(data, &networks); err != nil {
		return []domain.Program{}, nil
	}

	if len(networks) == 0 || len(networks[0].Items) == 0 {
		return []domain.Program{}, nil
	}

	dateStr := date.Format("2006-01-02")
	var epgItems []epgItem
	for _, item := range networks[0].Items {
		if item.Date == dateStr {
			epgItems = item.EPG
			break
		}
	}

	if len(epgItems) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(epgItems))
	for i, item := range epgItems {
		start, err := time.ParseInLocation("2006-01-02 15:04:05", item.StartDatetime, hkLocation)
		if err != nil {
			continue
		}

		stop := start.Add(30 * time.Minute)

		if i > 0 && len(programs) > 0 {
			programs[len(programs)-1].Stop = start.UTC()
		}

		var title, description string
		if channel.Lang == "en" {
			title = item.ProgrammeTitleEN
			description = item.EpisodeSynopsisEN
		} else {
			title = item.ProgrammeTitleTC
			description = item.EpisodeSynopsisTC
		}

		episodeNo, _ := strconv.Atoi(item.EpisodeNo)

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "mytvsuper.com",
			Title:       title,
			Description: description,
			Episode:     episodeNo,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		})
	}

	return programs, nil
}

type networkData struct {
	NetworkCode string     `json:"network_code"`
	Items       []dateItem `json:"item"`
}

type dateItem struct {
	Date string    `json:"date"`
	EPG  []epgItem `json:"epg"`
}

type epgItem struct {
	StartDatetime     string `json:"start_datetime"`
	ProgrammeTitleTC  string `json:"programme_title_tc"`
	ProgrammeTitleEN  string `json:"programme_title_en"`
	EpisodeSynopsisTC string `json:"episode_synopsis_tc"`
	EpisodeSynopsisEN string `json:"episode_synopsis_en"`
	EpisodeNo         string `json:"episode_no"`
}
