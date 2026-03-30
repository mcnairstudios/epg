package teleboy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

const apiKey = "e899f715940a209148f834702fc7f340b6b0496b62120b3ed9c9b3ec4d7dca00"

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

func (p *provider) Name() string        { return "teleboy.ch" }
func (p *provider) Description() string { return "Teleboy Switzerland EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "teleboy.ch")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	begin := date.Format("2006-01-02 15:04:05")
	end := date.Add(24 * time.Hour).Format("2006-01-02 15:04:05")

	reqURL := fmt.Sprintf("https://api.teleboy.ch/epg/broadcasts?begin=%s&end=%s&expand=flags,primary_image&station=%s",
		url.QueryEscape(begin), url.QueryEscape(end), channel.SiteID)

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-teleboy-apikey", apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from teleboy", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return []domain.Program{}, nil
	}

	if len(resp.Data.Items) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(resp.Data.Items))
	for _, item := range resp.Data.Items {
		start, err := time.Parse("2006-01-02T15:04:05-0700", item.Begin)
		if err != nil {
			continue
		}

		stop, err := time.Parse("2006-01-02T15:04:05-0700", item.End)
		if err != nil {
			continue
		}

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "teleboy.ch",
			Title:       item.Title,
			SubTitle:    item.Subtitle,
			Description: item.ShortDescription,
			Lang:        channel.Lang,
			Season:      item.SerieSeason,
			Episode:     item.SerieEpisode,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		}

		if item.PrimaryImage != nil && item.PrimaryImage.BasePath != "" && item.PrimaryImage.Hash != "" {
			p.Image = fmt.Sprintf("%steleboyteaser6/%s.jpg", item.PrimaryImage.BasePath, item.PrimaryImage.Hash)
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Data apiData `json:"data"`
}

type apiData struct {
	Total int       `json:"total"`
	Items []apiItem `json:"items"`
}

type apiItem struct {
	ID               int           `json:"id"`
	Begin            string        `json:"begin"`
	End              string        `json:"end"`
	Title            string        `json:"title"`
	Subtitle         string        `json:"subtitle"`
	ShortDescription string        `json:"short_description"`
	Year             int           `json:"year"`
	SerieSeason      int           `json:"serie_season"`
	SerieEpisode     int           `json:"serie_episode"`
	IMDBRating       float64       `json:"imdb_rating"`
	PrimaryImage     *primaryImage `json:"primary_image"`
}

type primaryImage struct {
	BasePath string `json:"base_path"`
	Hash     string `json:"hash"`
}
