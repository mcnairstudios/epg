package telenet

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
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

const (
	staticEndpoint = "https://static.spark.telenet.tv/eng/web/epg-service-lite/be"
	prodEndpoint   = "https://spark-prod-be.gnp.cloud.telenet.tv/eng/web/linear-service/v2"
	imageEndpoint  = "https://staticqbr-prod-be.gnp.cloud.telenet.tv/image-service"
)

func init() {
	plugin.Register(&provider{client: httpclient.Browser()})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "telenet.tv" }
func (p *provider) Description() string { return "Telenet TV Belgium EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "telenet.tv")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	day := date.Format("20060102")
	segments := []string{"000000", "060000", "120000", "180000"}

	var allEvents []Event
	for _, seg := range segments {
		url := fmt.Sprintf("%s/%s/events/segments/%s%s", staticEndpoint, channel.Lang, day, seg)
		events, err := p.fetchSegment(url, channel.SiteID)
		if err != nil {
			continue
		}
		allEvents = append(allEvents, events...)
	}

	programs := make([]domain.Program, 0, len(allEvents))
	for _, ev := range allEvents {
		detail := p.fetchDetail(ev.ID, channel.Lang)

		prog := domain.Program{
			Channel: channel.XmltvID,
			Source:  "telenet.tv",
			Title:   ev.Title,
			Start:   time.Unix(ev.StartTime, 0).UTC(),
			Stop:    time.Unix(ev.EndTime, 0).UTC(),
			Lang:    channel.Lang,
		}

		if ev.ID != "" {
			prog.Image = fmt.Sprintf("%s/intent/%s/posterTile", imageEndpoint, ev.ID)
		}

		if detail != nil {
			prog.Description = detail.LongDescription
			prog.SubTitle = detail.EpisodeName
			prog.Season = parseSeason(detail.SeasonNumber)
			prog.Episode = parseEpisode(detail.EpisodeNumber)
			if len(detail.Genres) > 0 {
				prog.Category = strings.Join(detail.Genres, ", ")
			}
		}

		programs = append(programs, prog)
	}

	return programs, nil
}

func (p *provider) fetchSegment(url string, siteID string) ([]Event, error) {
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
		return nil, fmt.Errorf("HTTP %d from telenet", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return ParseSegment(body, siteID)
}

func (p *provider) fetchDetail(eventID string, lang string) *ProgramDetail {
	if eventID == "" {
		return nil
	}

	url := fmt.Sprintf("%s/replayEvent/%s?returnLinearContent=true&language=%s", prodEndpoint, eventID, lang)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	detail, err := ParseDetail(body)
	if err != nil {
		return nil
	}
	return detail
}

func ParseSegment(data []byte, siteID string) ([]Event, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var seg segmentResponse
	if err := json.Unmarshal(data, &seg); err != nil {
		return nil, err
	}

	for _, entry := range seg.Entries {
		if entry.ChannelID == siteID {
			return entry.Events, nil
		}
	}

	return nil, nil
}

func ParseDetail(data []byte) (*ProgramDetail, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var detail ProgramDetail
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, err
	}

	return &detail, nil
}

func parseSeason(n int) int {
	if n == 0 {
		return 0
	}
	if len(fmt.Sprintf("%d", n)) > 2 {
		return 0
	}
	return n
}

func parseEpisode(n int) int {
	if n == 0 {
		return 0
	}
	if len(fmt.Sprintf("%d", n)) > 3 {
		return 0
	}
	return n
}

type segmentResponse struct {
	Duration int             `json:"duration"`
	Entries  []channelEntry  `json:"entries"`
}

type channelEntry struct {
	ChannelID string  `json:"channelId"`
	Events []Event `json:"events"`
}

type Event struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
}

type ProgramDetail struct {
	Title           string   `json:"title"`
	LongDescription string   `json:"longDescription"`
	EpisodeName     string   `json:"episodeName"`
	SeasonNumber    int      `json:"seasonNumber"`
	EpisodeNumber   int      `json:"episodeNumber"`
	Genres          []string `json:"genres"`
	Actors          []string `json:"actors"`
}

func (p *provider) Timezone() string { return "Europe/Brussels" }
