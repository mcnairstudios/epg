package tvplustr

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

var seasonEpisodeRe = regexp.MustCompile(`(\d+)\. Sezon - (\d+)\. Bölüm`)

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

func (p *provider) Name() string        { return "tvplus.com.tr" }
func (p *provider) Description() string { return "Turkcell TV+ Turkish IPTV provider" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvplus.com.tr")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	buildID, err := p.fetchBuildID()
	if err != nil {
		return nil, fmt.Errorf("failed to get build ID: %w", err)
	}

	channelID := strings.ReplaceAll(channel.SiteID, "/", "--")
	url := fmt.Sprintf(
		"https://tvplus.com.tr/_next/data/%s/%s/canli-tv/yayin-akisi/%s.json?title=%s",
		buildID, channel.Lang, channelID, channelID,
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
		return nil, fmt.Errorf("HTTP %d from tvplustr", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

func (p *provider) fetchBuildID() (string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://tvplus.com.tr/canli-tv/yayin-akisi", nil)
	if err != nil {
		return "", err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d from tvplustr", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`"buildId":"([^"]+)"`)
	matches := re.FindSubmatch(body)
	if len(matches) < 2 {
		return "", fmt.Errorf("build ID not found in page")
	}

	return string(matches[1]), nil
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	var response pageResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return []domain.Program{}, nil
	}

	dateStr := date.Format("2006-01-02")
	var programs []domain.Program

	for _, playbill := range response.PageProps.AllPlaybillList {
		if len(playbill) == 0 {
			continue
		}
		if !strings.HasPrefix(playbill[0].StartTime, dateStr) {
			continue
		}
		for _, schedule := range playbill {
			p := domain.Program{
				Channel:     channel.XmltvID,
				Source:      "tvplus.com.tr",
				Title:       schedule.Name,
				Description: schedule.Introduce,
				Category:    schedule.Genres,
				Image:       schedule.Picture,
				Lang:        channel.Lang,
			}

			if schedule.StartTime != "" {
				if t, err := parseTVPlusTime(schedule.StartTime); err == nil {
					p.Start = t
				}
			}
			if schedule.EndTime != "" {
				if t, err := parseTVPlusTime(schedule.EndTime); err == nil {
					p.Stop = t
				}
			}

			if schedule.SeasonInfo != "" {
				matches := seasonEpisodeRe.FindStringSubmatch(schedule.SeasonInfo)
				if len(matches) == 3 {
					p.Season, _ = strconv.Atoi(matches[1])
					p.Episode, _ = strconv.Atoi(matches[2])
				}
			}

			programs = append(programs, p)
		}
	}

	if programs == nil {
		programs = []domain.Program{}
	}

	return programs, nil
}

func parseTVPlusTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	formats := []string{
		"2006-01-02 15:04:05 UTC-07:00",
		"2006-01-02 15:04:05 UTC+07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}

type pageResponse struct {
	PageProps pageProps `json:"pageProps"`
}

type pageProps struct {
	AllPlaybillList [][]schedule `json:"allPlaybillList"`
}

type schedule struct {
	Name       string `json:"name"`
	StartTime  string `json:"starttime"`
	EndTime    string `json:"endtime"`
	Genres     string `json:"genres"`
	Picture    string `json:"picture"`
	Introduce  string `json:"introduce"`
	SeasonInfo string `json:"seasonInfo"`
}
