package virginmedia

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

func (p *provider) Name() string        { return "virgintvgo.virginmedia.com" }
func (p *provider) Description() string { return "Virgin TV Go UK EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "virgintvgo.virginmedia.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://staticqbr-prod-gb.gnp.cloud.virgintvgo.virginmedia.com/eng/web/epg-service-lite/gb/en/events/segments/%s000000",
		date.Format("20060102"),
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
		return nil, fmt.Errorf("HTTP %d from virginmedia", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return ParseSegment(body, channel)
}

func ParseSegment(data []byte, channel domain.Channel) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	var seg segmentResponse
	if err := json.Unmarshal(data, &seg); err != nil {
		return []domain.Program{}, nil
	}

	var eventIDs []string
	for _, entry := range seg.Entries {
		if entry.ChannelID != channel.SiteID {
			continue
		}
		for _, ev := range entry.Events {
			eventIDs = append(eventIDs, ev.ID)
		}
	}

	var programs []domain.Program
	for _, entry := range seg.Entries {
		if entry.ChannelID != channel.SiteID {
			continue
		}
		for _, ev := range entry.Events {
			programs = append(programs, domain.Program{
				Channel: channel.XmltvID,
				Source:  "virgintvgo.virginmedia.com",
				Title:   ev.Title,
				Lang:    channel.Lang,
				Start:   time.Unix(ev.StartTime, 0).UTC(),
				Stop:    time.Unix(ev.EndTime, 0).UTC(),
			})
		}
	}

	if programs == nil {
		programs = []domain.Program{}
	}

	return programs, nil
}

func ParseProgram(data []byte, channel domain.Channel) (domain.Program, error) {
	var item programDetail
	if err := json.Unmarshal(data, &item); err != nil {
		return domain.Program{}, err
	}

	p := domain.Program{
		Channel:     channel.XmltvID,
		Source:      "virgintvgo.virginmedia.com",
		Title:       item.Title,
		SubTitle:    item.EpisodeName,
		Description: item.LongDescription,
		Season:      item.SeasonNumber,
		Episode:     item.EpisodeNumber,
		Lang:        channel.Lang,
		Start:       time.Unix(item.StartTime, 0).UTC(),
		Stop:        time.Unix(item.EndTime, 0).UTC(),
	}

	if p.Description == "" {
		p.Description = item.ShortDescription
	}

	if len(item.Genres) > 0 {
		p.Category = item.Genres[0]
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

type programDetail struct {
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
