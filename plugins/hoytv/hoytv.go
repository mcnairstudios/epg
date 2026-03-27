package hoytv

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

var hongKongTZ = time.FixedZone("Asia/Hong_Kong", 8*60*60)

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

func (p *provider) Name() string        { return "hoy.tv" }
func (p *provider) Description() string { return "HOY TV Hong Kong" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "hoy.tv")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()
	url := fmt.Sprintf("https://epg-file.hoy.tv/hoy/OTT%s%s.xml", channel.SiteID, d.Format("20060102"))

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
	var guide programGuide
	if err := xml.Unmarshal(data, &guide); err != nil {
		return nil, err
	}

	d := date.Truncate(24 * time.Hour).UTC()
	programs := make([]domain.Program, 0)

	for _, item := range guide.Channel.EpgItems {
		start, err := time.ParseInLocation("2006-01-02 15:04:05", item.EpgStartDateTime, hongKongTZ)
		if err != nil {
			continue
		}

		startUTC := start.UTC()
		if startUTC.Year() != d.Year() || startUTC.Month() != d.Month() || startUTC.Day() != d.Day() {
			hkDate := start
			if hkDate.Year() != d.Year() || hkDate.YearDay() != d.YearDay() {
				continue
			}
		}

		stop, err := time.ParseInLocation("2006-01-02 15:04:05", item.EpgEndDateTime, hongKongTZ)
		if err != nil {
			continue
		}

		title := item.ComScore.NsStPr
		if item.EpgOtherInfo != "" {
			title += item.EpgOtherInfo
		}

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "hoy.tv",
			Title:       title,
			Description: item.EpisodeInfo.EpisodeLongDescription,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		}

		epIdx, _ := strconv.Atoi(item.EpisodeInfo.EpisodeIndex)
		if epIdx > 0 {
			p.SubTitle = fmt.Sprintf("第%s集", item.EpisodeInfo.EpisodeIndex)
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type programGuide struct {
	XMLName xml.Name `xml:"ProgramGuide"`
	Channel xmlChannel `xml:"Channel"`
}

type xmlChannel struct {
	EpgItems []epgItem `xml:"EpgItem"`
}

type epgItem struct {
	EpgStartDateTime string      `xml:"EpgStartDateTime"`
	EpgEndDateTime   string      `xml:"EpgEndDateTime"`
	EpgOtherInfo     string      `xml:"EpgOtherInfo"`
	ComScore         comScore    `xml:"ComScore"`
	EpisodeInfo      episodeInfo `xml:"EpisodeInfo"`
}

type comScore struct {
	NsStPr string `xml:"ns_st_pr"`
}

type episodeInfo struct {
	EpisodeIndex           string `xml:"EpisodeIndex"`
	EpisodeLongDescription string `xml:"EpisodeLongDescription"`
}
