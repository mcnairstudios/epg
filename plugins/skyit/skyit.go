package skyit

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

func (p *provider) Name() string        { return "guidatv.sky.it" }
func (p *provider) Description() string { return "Sky Italia TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "guidatv.sky.it")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	parts := strings.SplitN(channel.SiteID, "#", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid site_id %q: expected env#id", channel.SiteID)
	}
	env := parts[0]
	id := parts[1]

	start := date.Truncate(24 * time.Hour).UTC()
	end := start.Add(24 * time.Hour)

	url := fmt.Sprintf(
		"https://apid.sky.it/gtv/v1/events?from=%s&to=%s&pageSize=999&pageNum=0&env=%s&channels=%s",
		start.Format("2006-01-02T15:04:05Z"),
		end.Format("2006-01-02T15:04:05Z"),
		env,
		id,
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
		return nil, fmt.Errorf("HTTP %d from skyit", resp.StatusCode)
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

	programs := make([]domain.Program, 0, len(response.Events))
	for _, item := range response.Events {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "guidatv.sky.it",
			Title:       item.EventTitle,
			Description: item.EventSynopsis,
			Lang:        channel.Lang,
			Season:      parseSeason(item),
			Episode:     parseEpisode(item),
		}

		if item.StartTime != "" {
			if t, err := time.Parse(time.RFC3339, item.StartTime); err == nil {
				p.Start = t
			}
		}
		if item.EndTime != "" {
			if t, err := time.Parse(time.RFC3339, item.EndTime); err == nil {
				p.Stop = t
			}
		}

		p.Category = parseCategory(item)
		p.Image = parseImage(item)

		programs = append(programs, p)
	}

	return programs, nil
}

func parseCategory(item event) string {
	genre := item.Content.Genre.Name
	subgenre := item.Content.Subgenre.Name
	if genre != "" && subgenre != "" {
		return genre + "/" + subgenre
	}
	return genre
}

func parseImage(item event) string {
	for _, img := range item.Content.ImagesMap {
		if img.Key == "cover" && img.Img.URL != "" {
			return "https://guidatv.sky.it" + img.Img.URL
		}
	}
	return ""
}

func parseSeason(item event) int {
	s := item.Content.SeasonNumber
	if s == 0 || s > 99 {
		return 0
	}
	return s
}

func parseEpisode(item event) int {
	e := item.Content.EpisodeNumber
	if e == 0 || e > 999 {
		return 0
	}
	return e
}

type apiResponse struct {
	Events []event `json:"events"`
}

type event struct {
	EventTitle   string  `json:"eventTitle"`
	EventSynopsis string `json:"eventSynopsis"`
	StartTime    string  `json:"starttime"`
	EndTime      string  `json:"endtime"`
	Content      content `json:"content"`
}

type content struct {
	URL           string     `json:"url"`
	SeasonNumber  int        `json:"seasonNumber"`
	EpisodeNumber int        `json:"episodeNumber"`
	Genre         genreInfo  `json:"genre"`
	Subgenre      genreInfo  `json:"subgenre"`
	ImagesMap     []imageMap `json:"imagesMap"`
}

type genreInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type imageMap struct {
	Key string   `json:"key"`
	Img imageURL `json:"img"`
}

type imageURL struct {
	URL string `json:"url"`
}
