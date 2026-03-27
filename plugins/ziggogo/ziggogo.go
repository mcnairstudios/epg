package ziggogo

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

func (p *provider) Name() string        { return "ziggogo.tv" }
func (p *provider) Description() string { return "Ziggo GO Netherlands EPG (156 channels)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "ziggogo.tv")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	day := date.Truncate(24 * time.Hour).UTC()
	dateStr := day.Format("20060102")

	segments := []string{"00", "06", "12", "18"}
	var allEvents []segmentEvent

	for _, seg := range segments {
		url := fmt.Sprintf(
			"https://static.spark.ziggogo.tv/eng/web/epg-service-lite/nl/en/events/segments/%s%s0000",
			dateStr, seg,
		)

		data, err := p.doGet(url)
		if err != nil {
			continue
		}

		var resp segmentResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			continue
		}

		for _, entry := range resp.Entries {
			if entry.ChannelID != channel.SiteID {
				continue
			}
			allEvents = append(allEvents, entry.Events...)
		}
	}

	var programs []domain.Program
	for _, evt := range allEvents {
		url := fmt.Sprintf(
			"https://spark-prod-nl.gnp.cloud.ziggogo.tv/eng/web/linear-service/v2/replayEvent/%s?returnLinearContent=true&forceLinearResponse=true&language=nl",
			evt.ID,
		)

		data, err := p.doGet(url)
		if err != nil {
			continue
		}

		prog, err := Parse(data, channel)
		if err != nil {
			continue
		}

		programs = append(programs, prog)
	}

	return programs, nil
}

func (p *provider) doGet(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func Parse(data []byte, channel domain.Channel) (domain.Program, error) {
	var detail detailResponse
	if err := json.Unmarshal(data, &detail); err != nil {
		return domain.Program{}, err
	}

	p := domain.Program{
		Channel:     channel.XmltvID,
		Source:      "ziggogo.tv",
		Title:       detail.Title,
		SubTitle:    detail.EpisodeName,
		Season:      detail.SeasonNumber,
		Episode:     detail.EpisodeNumber,
		Lang:        channel.Lang,
		Start:       time.Unix(detail.StartTime, 0).UTC(),
		Stop:        time.Unix(detail.EndTime, 0).UTC(),
	}

	if detail.LongDescription != "" {
		p.Description = detail.LongDescription
	} else {
		p.Description = detail.ShortDescription
	}

	if len(detail.Genres) > 0 {
		p.Category = strings.Join(detail.Genres, ", ")
	}

	return p, nil
}

type segmentResponse struct {
	Duration int             `json:"duration"`
	Entries  []segmentEntry  `json:"entries"`
}

type segmentEntry struct {
	ChannelID string         `json:"channelId"`
	Events    []segmentEvent `json:"events"`
}

type segmentEvent struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
}

type detailResponse struct {
	Title            string   `json:"title"`
	EpisodeName      string   `json:"episodeName"`
	ShortDescription string   `json:"shortDescription"`
	LongDescription  string   `json:"longDescription"`
	SeasonNumber     int      `json:"seasonNumber"`
	EpisodeNumber    int      `json:"episodeNumber"`
	StartTime        int64    `json:"startTime"`
	EndTime          int64    `json:"endTime"`
	Genres           []string `json:"genres"`
	Actors           []string `json:"actors"`
	Directors        []string `json:"directors"`
	Producers        []string `json:"producers"`
	CountryOfOrigin  string   `json:"countryOfOrigin"`
	ProductionDate   string   `json:"productionDate"`
}
