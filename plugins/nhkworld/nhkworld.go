package nhkworld

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

var seoulTZ = time.FixedZone("Asia/Seoul", 9*60*60)

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

func (p *provider) Name() string        { return "nhkworldpremium.com" }
func (p *provider) Description() string { return "NHK World Premium Japanese TV service" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "nhkworld-en", Name: "NHK World Premium", Source: "nhkworldpremium.com", SiteID: "#", Lang: "en", XmltvID: "NHKWorldPremium.jp"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://nhkworldpremium.com/backend/api/v1/front/episodes?lang=%s", channel.Lang)

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
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	if response.Item == nil || response.Item.Episodes == nil {
		return []domain.Program{}, nil
	}

	dateStr := date.Truncate(24 * time.Hour).UTC().Format("2006-01-02")

	programs := make([]domain.Program, 0)
	for _, ep := range response.Item.Episodes {
		if !strings.HasPrefix(ep.Schedule, dateStr) {
			continue
		}

		start, err := time.ParseInLocation("2006-01-02 15:04:05", ep.Schedule, seoulTZ)
		if err != nil {
			continue
		}

		duration := parseDuration(ep.Period)
		stop := start.Add(duration)

		subtitle := ""
		if ep.EpisodeTitle != nil {
			subtitle = *ep.EpisodeTitle
		}

		p := domain.Program{
			Channel:  channel.XmltvID,
			Source:   "nhkworldpremium.com",
			Title:    ep.ProgramTitle,
			SubTitle: subtitle,
			Lang:     channel.Lang,
			Start:    start.UTC(),
			Stop:     stop.UTC(),
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func parseDuration(period string) time.Duration {
	parts := strings.Split(period, ":")
	if len(parts) != 3 {
		return 0
	}

	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	s, _ := strconv.Atoi(parts[2])

	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s)*time.Second
}

type apiResponse struct {
	Item *itemWrapper `json:"item"`
}

type itemWrapper struct {
	Episodes []episodeItem `json:"episodes"`
}

type episodeItem struct {
	ProgramTitle string  `json:"programTitle"`
	EpisodeTitle *string `json:"episodeTitle"`
	Schedule     string  `json:"schedule"`
	Period       string  `json:"period"`
}
