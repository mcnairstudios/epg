package mediaset

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
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

func (p *provider) Name() string        { return "mediasetinfinity.mediaset.it" }
func (p *provider) Description() string { return "Mediaset Infinity Italian TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "mediasetinfinity.mediaset.it")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour).UTC()
	end := start.Add(24 * time.Hour)

	url := fmt.Sprintf(
		"https://api-ott-prod-fe.mediaset.net/PROD/play/feed/allListingFeedEpg/v2.0?byListingTime=%d~%d&byCallSign=%s",
		start.UnixMilli(),
		end.UnixMilli(),
		channel.SiteID,
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
		return nil, fmt.Errorf("HTTP %d from mediaset", resp.StatusCode)
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

	if response.Response == nil || len(response.Response.Entries) == 0 || len(response.Response.Entries[0].Listings) == 0 {
		return []domain.Program{}, nil
	}

	listings := response.Response.Entries[0].Listings
	programs := make([]domain.Program, 0, len(listings))

	for _, listing := range listings {
		if listing.Program.Title == "" || listing.StartTime == 0 || listing.EndTime == 0 {
			continue
		}

		title := listing.EpgTitle
		subTitle := listing.Program.Title
		if title == "" {
			title = subTitle
			subTitle = ""
		} else if title == subTitle {
			subTitle = ""
		}

		season := parseSeason(listing.ShortDescription)
		episode := parseEpisode(listing.ShortDescription)
		if episode > 0 && season == 0 {
			season = 0
		}

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "mediasetinfinity.mediaset.it",
			Title:       title,
			SubTitle:    subTitle,
			Description: listing.Program.Description,
			Category:    listing.Program.SkyGenre,
			Season:      season,
			Episode:     episode,
			Image:       getMaxResolutionImage(listing.Program.Thumbnails),
			Lang:        channel.Lang,
			Start:       time.UnixMilli(listing.StartTime).UTC(),
			Stop:        time.UnixMilli(listing.EndTime).UTC(),
		}

		programs = append(programs, p)
	}

	return programs, nil
}

var seasonRegex = regexp.MustCompile(`S(\d+)\s`)
var episodeRegex = regexp.MustCompile(`Ep(\d+)\s`)

func parseSeason(desc string) int {
	if desc == "" {
		return 0
	}
	m := seasonRegex.FindStringSubmatch(desc)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

func parseEpisode(desc string) int {
	if desc == "" {
		return 0
	}
	m := episodeRegex.FindStringSubmatch(desc)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

func getMaxResolutionImage(thumbnails map[string]thumbnail) string {
	if len(thumbnails) == 0 {
		return ""
	}

	best := make(map[string]thumbnail)
	for key, t := range thumbnails {
		typ := strings.Split(key, "-")[0]
		if existing, ok := best[typ]; !ok || t.Width*t.Height > existing.Width*existing.Height {
			best[typ] = t
		}
	}

	if t, ok := best["image_keyframe_poster"]; ok {
		return t.URL
	}
	if t, ok := best["image_header_poster"]; ok {
		return t.URL
	}
	return ""
}

type apiResponse struct {
	Response *responseBody `json:"response"`
}

type responseBody struct {
	Entries []entry `json:"entries"`
}

type entry struct {
	Listings []listing `json:"listings"`
}

type listing struct {
	StartTime        int64   `json:"startTime"`
	EndTime          int64   `json:"endTime"`
	EpgTitle         string  `json:"mediasetlisting$epgTitle"`
	ShortDescription string  `json:"mediasetlisting$shortDescription"`
	Program          program `json:"program"`
}

type program struct {
	Title       string               `json:"title"`
	Description string               `json:"description"`
	SkyGenre    string               `json:"mediasetprogram$skyGenre"`
	Thumbnails  map[string]thumbnail `json:"thumbnails"`
}

type thumbnail struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func (p *provider) Timezone() string { return "Europe/Rome" }
