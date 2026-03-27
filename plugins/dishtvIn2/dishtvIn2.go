package dishtvIn2

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
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

var kolkata = time.FixedZone("IST", 5*3600+30*60)

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

func (p *provider) Name() string        { return "dishtv.in" }
func (p *provider) Description() string { return "DishTV India satellite television" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "dishtv.in")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	token, err := p.fetchToken()
	if err != nil {
		return nil, fmt.Errorf("fetching token: %w", err)
	}

	body, err := json.Marshal(requestBody{
		AllowPastEvents: true,
		ChannelID:       channel.SiteID,
		Date:            date.In(kolkata).Format("02/01/2006"),
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, "https://epg.mysmartstick.com/dishtv/api/v1/epg/entities/programs", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel)
}

func (p *provider) fetchToken() (string, error) {
	req, err := http.NewRequest(http.MethodPost, "https://www.dishtv.in/services/epg/signin", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("Referer", "https://www.dishtv.in/channel-guide.html")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var session sessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return "", err
	}

	return session.Token, nil
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var items []apiItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "dishtv.in",
			Lang:    channel.Lang,
		}

		if en, ok := item.Regional["english"]; ok {
			p.Title = en.Title
			p.Description = en.Desc
			p.Category = en.Genre
		} else {
			p.Title = item.Title
			p.Description = item.Desc
			p.Category = item.Genre
		}

		if item.EpisodeNum != "" {
			if n, err := strconv.Atoi(item.EpisodeNum); err == nil {
				p.Episode = n
			}
		}

		if item.Images.Landscape != nil {
			if v, ok := item.Images.Landscape["1280x720"]; ok {
				p.Image = v
			}
		}

		p.Start = parseIST(item.Start)
		p.Stop = parseIST(item.Stop)

		programs = append(programs, p)
	}

	return programs, nil
}

func parseIST(s string) time.Time {
	s = strings.TrimSuffix(s, "Z")
	t, err := time.ParseInLocation("2006-01-02T15:04:05", s, kolkata)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

type requestBody struct {
	AllowPastEvents bool   `json:"allowPastEvents"`
	ChannelID       string `json:"channelid"`
	Date            string `json:"date"`
}

type sessionResponse struct {
	Token string `json:"token"`
}

type apiItem struct {
	Title          string                `json:"title"`
	Start          string                `json:"start"`
	Stop           string                `json:"stop"`
	Desc           string                `json:"desc"`
	Genre          string                `json:"genre"`
	ProductionYear string                `json:"productionyear"`
	ProgrammeURL   string                `json:"programmeurl"`
	EpisodeNum     string                `json:"episode-num"`
	Images         imageSet              `json:"images"`
	Credits        credits               `json:"credits"`
	Regional       map[string]regionData `json:"regional"`
}

type imageSet struct {
	Landscape map[string]string `json:"landscape"`
}

type credits struct {
	Actors    []string `json:"actors"`
	Directors []string `json:"directors"`
	Producers []string `json:"producers"`
}

type regionData struct {
	LanguageCode string `json:"languagecode"`
	Title        string `json:"title"`
	Desc         string `json:"desc"`
	Genre        string `json:"genre"`
}
