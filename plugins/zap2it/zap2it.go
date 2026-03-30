package zap2it

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		sleep:  time.Sleep,
	})
}

type provider struct {
	client httpClient
	sleep  func(time.Duration)
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "zap2it.com" }
func (p *provider) Description() string { return "Zap2it US/Canada TV listings via Gracenote" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "zap2it.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	parts := strings.SplitN(channel.SiteID, "/", 6)
	if len(parts) != 6 {
		return nil, fmt.Errorf("invalid site_id format: %s", channel.SiteID)
	}

	device := parts[0]
	lineupID := parts[1]
	headendID := parts[2]
	countryCode := parts[3]
	postalCode := parts[4]
	prgsvcid := parts[5]

	timestamp := strconv.FormatInt(date.Truncate(24*time.Hour).UTC().Unix(), 10)

	body := apiRequest{
		LineupID:           lineupID,
		IsSSLinkNavigation: "true",
		Timespan:           "336",
		Timestamp:          timestamp,
		Prgsvcid:           prgsvcid,
		HeadendID:          headendID,
		CountryCode:        countryCode,
		PostalCode:         postalCode,
		Device:             device,
		UserID:             "-",
		Aid:                "orbebb",
		DSTUTCOffset:       "-240",
		STDUTCOffset:       "-300",
		DSTStart:           usDSTStart(date.Year()),
		DSTEnd:             usDSTEnd(date.Year()),
		LanguageCode:       "en-us",
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	var data []byte
	backoff := time.Second
	const maxRetries = 3

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest(http.MethodPost, "https://tvlistings.gracenote.com/api/sslgrid", bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")

		resp, err := p.client.Do(req)
		if err != nil {
			if attempt < maxRetries {
				p.sleep(backoff)
				backoff *= 2
				continue
			}
			return nil, err
		}

		data, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			if attempt < maxRetries {
				p.sleep(backoff)
				backoff *= 2
				continue
			}
			return nil, nil
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP %d from zap2it", resp.StatusCode)
		}

		break
	}

	return Parse(data, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response map[string][]apiEntry
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil
	}

	var programs []domain.Program
	for _, entries := range response {
		for _, item := range entries {
			p := domain.Program{
				Channel: channel.XmltvID,
				Source:  "zap2it.com",
				Title:   item.Program.Title,
				Lang:    channel.Lang,
				Start:   time.Unix(item.StartTime, 0).UTC(),
				Stop:    time.Unix(item.EndTime, 0).UTC(),
			}

			if item.Program.EpisodeTitle != "" {
				p.SubTitle = item.Program.EpisodeTitle
			}
			if item.Program.ShortDesc != "" {
				p.Description = item.Program.ShortDesc
			}
			if len(item.Program.Genres) > 0 {
				p.Category = item.Program.Genres[0].Name
			}
			if item.Thumbnail != "" {
				p.Image = fmt.Sprintf("https://zap2it.tmsimg.com/assets/%s.jpg", item.Thumbnail)
			}
			if item.Program.Season != "" {
				if n, err := strconv.Atoi(item.Program.Season); err == nil {
					p.Season = n
				}
			}
			if item.Program.Episode != "" {
				if n, err := strconv.Atoi(item.Program.Episode); err == nil {
					p.Episode = n
				}
			}

			programs = append(programs, p)
		}
	}

	return programs, nil
}

type apiRequest struct {
	LineupID           string `json:"lineupId"`
	IsSSLinkNavigation string `json:"IsSSLinkNavigation"`
	Timespan           string `json:"timespan"`
	Timestamp          string `json:"timestamp"`
	Prgsvcid           string `json:"prgsvcid"`
	HeadendID          string `json:"headendId"`
	CountryCode        string `json:"countryCode"`
	PostalCode         string `json:"postalCode"`
	Device             string `json:"device"`
	UserID             string `json:"userId"`
	Aid                string `json:"aid"`
	DSTUTCOffset       string `json:"DSTUTCOffset"`
	STDUTCOffset       string `json:"STDUTCOffset"`
	DSTStart           string `json:"DSTStart"`
	DSTEnd             string `json:"DSTEnd"`
	LanguageCode       string `json:"languagecode"`
}

type apiEntry struct {
	Program   programInfo `json:"program"`
	StartTime int64       `json:"startTime"`
	EndTime   int64       `json:"endTime"`
	Thumbnail string      `json:"thumbnail"`
	Rating    string      `json:"rating"`
}

type programInfo struct {
	Title        string  `json:"title"`
	EpisodeTitle string  `json:"episodeTitle"`
	ShortDesc    string  `json:"shortDesc"`
	Genres       []genre `json:"genres"`
	Season       string  `json:"season"`
	Episode      string  `json:"episode"`
	ReleaseYear  string  `json:"releaseYear"`
}

type genre struct {
	Name string `json:"name"`
}


// usDSTStart returns the US DST start date (second Sunday in March) for the given year.
func usDSTStart(year int) string {
	t := time.Date(year, time.March, 1, 2, 0, 0, 0, time.UTC)
	sundays := 0
	for {
		if t.Weekday() == time.Sunday {
			sundays++
			if sundays == 2 {
				break
			}
		}
		t = t.AddDate(0, 0, 1)
	}
	return t.Format("2006-01-02T15:04Z")
}

// usDSTEnd returns the US DST end date (first Sunday in November) for the given year.
func usDSTEnd(year int) string {
	t := time.Date(year, time.November, 1, 2, 0, 0, 0, time.UTC)
	for t.Weekday() != time.Sunday {
		t = t.AddDate(0, 0, 1)
	}
	return t.Format("2006-01-02T15:04Z")
}
