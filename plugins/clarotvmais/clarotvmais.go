package clarotvmais

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

var saoPauloTZ *time.Location

func init() {
	var err error
	saoPauloTZ, err = time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		saoPauloTZ = time.FixedZone("America/Sao_Paulo", -3*60*60)
	}
}

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

func (p *provider) Name() string        { return "clarotvmais.com.br" }
func (p *provider) Description() string { return "Claro TV+ Brazil" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "clarotvmais.com.br")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	// JS uses dayjs(date).startOf('day') and endOf('day') in America/Sao_Paulo timezone
	spDate := date.In(saoPauloTZ)
	startOfDay := time.Date(spDate.Year(), spDate.Month(), spDate.Day(), 0, 0, 0, 0, saoPauloTZ)
	endOfDay := time.Date(spDate.Year(), spDate.Month(), spDate.Day(), 23, 59, 59, 0, saoPauloTZ)

	url := fmt.Sprintf(
		"https://www.clarotvmais.com.br/avsclient/1.2/epg/livechannels?types=&channelIds=%s&startTime=%d&endTime=%d&location=SAO%%20PAULO,AMAZONAS&channel=PCTV",
		channel.SiteID, startOfDay.Unix(), endOfDay.Unix(),
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
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
		return []domain.Program{}, nil
	}

	if len(response.Response.LiveChannels) == 0 {
		return []domain.Program{}, nil
	}

	schedules := response.Response.LiveChannels[0].Schedules
	if len(schedules) <= 1 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(schedules))
	for _, s := range schedules {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "clarotvmais.com.br",
			Title:       s.Title,
			Description: s.Description,
			Lang:        channel.Lang,
			Start:       time.Unix(s.StartTime, 0).UTC(),
			Stop:        time.Unix(s.EndTime, 0).UTC(),
		}

		if s.Image != "" {
			p.Image = strings.ReplaceAll(s.Image, "{{image-size-placeholder}}", "420_236")
		}
		if s.SeasonNumber > 0 {
			p.Season = s.SeasonNumber
		}
		if s.EpisodeNumber > 0 {
			p.Episode = s.EpisodeNumber
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Response responseData `json:"response"`
}

type responseData struct {
	LiveChannels []liveChannel `json:"liveChannels"`
}

type liveChannel struct {
	Schedules []scheduleItem `json:"schedules"`
}

type scheduleItem struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	Image         string `json:"image"`
	StartTime     int64  `json:"startTime"`
	EndTime       int64  `json:"endTime"`
	SeasonNumber  int    `json:"seasonNumber"`
	EpisodeNumber int    `json:"episodeNumber"`
}
