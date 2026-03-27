package trueid

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "tv.trueid.net" }
func (p *provider) Description() string { return "TrueID TV Thailand EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tv.trueid.net")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	buildID, err := p.fetchBuildID()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(
		"https://tv.trueid.net/_next/data/%s/th-%s.json?channelSlug=%s&path=%s",
		buildID, channel.Lang, channel.SiteID, channel.SiteID,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func (p *provider) fetchBuildID() (string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://tv.trueid.net/th-en", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return ParseBuildID(body)
}

func ParseBuildID(body []byte) (string, error) {
	var nextData struct {
		BuildID string `json:"buildId"`
	}

	start := indexOf(body, []byte(`{"props":`))
	if start < 0 {
		return "", fmt.Errorf("could not find __NEXT_DATA__")
	}

	end := indexOf(body[start:], []byte("</script>"))
	if end < 0 {
		return "", fmt.Errorf("could not find end of __NEXT_DATA__")
	}

	if err := json.Unmarshal(body[start:start+end], &nextData); err != nil {
		return "", err
	}

	return nextData.BuildID, nil
}

func indexOf(data, sub []byte) int {
	for i := range data {
		if i+len(sub) > len(data) {
			return -1
		}
		match := true
		for j := range sub {
			if data[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response pageResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return []domain.Program{}, nil
	}

	items := response.PageProps.EpgList
	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "tv.trueid.net",
			Title:   item.Title,
			Lang:    channel.Lang,
		}

		descKey := fmt.Sprintf("synopsis_%s", channel.Lang)
		if item.Info != nil {
			if desc, ok := item.Info[descKey]; ok {
				if s, ok := desc.(string); ok && s != "." {
					p.Description = s
				}
			}
			if img, ok := item.Info["image"]; ok {
				if s, ok := img.(string); ok {
					p.Image = s
				}
			}
		}

		if item.StartDate != "" {
			if t, err := time.Parse(time.RFC3339Nano, item.StartDate); err == nil {
				p.Start = t.UTC()
			} else if t, err := time.Parse("2006-01-02T15:04:05.000Z", item.StartDate); err == nil {
				p.Start = t.UTC()
			}
		}
		if item.EndDate != "" {
			if t, err := time.Parse(time.RFC3339Nano, item.EndDate); err == nil {
				p.Stop = t.UTC()
			} else if t, err := time.Parse("2006-01-02T15:04:05.000Z", item.EndDate); err == nil {
				p.Stop = t.UTC()
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type pageResponse struct {
	PageProps pageProps `json:"pageProps"`
}

type pageProps struct {
	EpgList []epgItem `json:"epgList"`
}

type epgItem struct {
	Title     string                 `json:"title"`
	StartDate string                 `json:"start_date"`
	EndDate   string                 `json:"end_date"`
	Info      map[string]interface{} `json:"info"`
}
