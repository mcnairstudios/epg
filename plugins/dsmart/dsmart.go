package dsmart

import (
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
	})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "dsmart.com.tr" }
func (p *provider) Description() string { return "D-Smart Turkey" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "dsmart.com.tr")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()
	url := fmt.Sprintf(
		"https://www.dsmart.com.tr/api/v1/public/epg/schedules?page=1&limit=10&day=%s",
		d.Format("2006-01-02"),
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
		return nil, fmt.Errorf("HTTP %d from dsmart", resp.StatusCode)
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

	if resp.Data == nil || resp.Data.Channels == nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0)
	for _, ch := range resp.Data.Channels {
		if ch.ID != channel.SiteID {
			continue
		}
		if len(ch.Schedule) == 0 {
			continue
		}

		var dayStart time.Time
		var ofs float64
		for i, s := range ch.Schedule {
			baseDate, err := time.Parse(time.RFC3339, s.Day)
			if err != nil {
				continue
			}
			startDate, err := time.Parse(time.RFC3339, s.StartDate)
			if err != nil {
				continue
			}

			if i == 0 {
				dayStart = startDate
				combined := baseDate.Format("2006-01-02T") + startDate.Format("15:04:05Z")
				ct, err := time.Parse(time.RFC3339, combined)
				if err != nil {
					continue
				}
				ofs = ct.Sub(baseDate).Seconds()
			}

			delta := startDate.Sub(dayStart).Seconds()
			start := baseDate.Add(time.Duration(ofs+delta) * time.Second)
			dur := parseDuration(s.Duration)
			stop := start.Add(dur)

			p := domain.Program{
				Channel:     channel.XmltvID,
				Source:      "dsmart.com.tr",
				Title:       s.ProgramName,
				Description: s.Description,
				Lang:        channel.Lang,
				Start:       start.UTC(),
				Stop:        stop.UTC(),
			}

			if s.Genre != "" && strings.Contains(s.Genre, "/") {
				parts := strings.Split(s.Genre, "/")
				cats := make([]string, 0, len(parts))
				for _, g := range parts {
					g = strings.TrimSpace(g)
					if len(g) > 0 {
						cats = append(cats, strings.ToUpper(g[:1])+g[1:])
					}
				}
				if len(cats) > 0 {
					p.Category = strings.Join(cats, "/")
				}
			}

			programs = append(programs, p)
		}
	}

	return programs, nil
}

func parseDuration(s string) time.Duration {
	durStr := s
	if strings.Contains(s, ",") {
		parts := strings.SplitN(s, ",", 2)
		durStr = strings.TrimSpace(parts[1])
	}

	parts := strings.Split(durStr, ":")
	if len(parts) != 3 {
		return 0
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	sec, _ := strconv.Atoi(parts[2])
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second
}

type apiResponse struct {
	Data *responseData `json:"data"`
}

type responseData struct {
	Total    int       `json:"total"`
	Channels []channel `json:"channels"`
}

type channel struct {
	ID       string     `json:"_id"`
	Name     string     `json:"channel_name"`
	Schedule []schedule `json:"schedule"`
}

type schedule struct {
	Day         string `json:"day"`
	StartDate   string `json:"start_date"`
	Duration    string `json:"duration"`
	ProgramName string `json:"program_name"`
	Description string `json:"description"`
	Genre       string `json:"genre"`
}
