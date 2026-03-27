package programetvro

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
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

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

func (p *provider) Name() string        { return "programetv.ro" }
func (p *provider) Description() string { return "Romanian TV schedule from programetv.ro" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "programetv.ro")
}

var daysOfWeek = map[int]string{
	0: "duminica",
	1: "luni",
	2: "marti",
	3: "miercuri",
	4: "joi",
	5: "vineri",
	6: "sambata",
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Truncate(24 * time.Hour).UTC()
	dayName := daysOfWeek[int(d.Weekday())]

	u := fmt.Sprintf(
		"https://www.programetv.ro/program-tv/%s/%s/",
		channel.SiteID,
		dayName,
	)

	var body []byte
	backoff := time.Second
	const maxRetries = 3

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "ro,en;q=0.5")

		resp, err := p.client.Do(req)
		if err != nil {
			if attempt < maxRetries {
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			return nil, err
		}

		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			if attempt < maxRetries {
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			return nil, fmt.Errorf("HTTP %d after %d retries for channel %s", resp.StatusCode, maxRetries, channel.SiteID)
		}

		if resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP %d for channel %s", resp.StatusCode, channel.SiteID)
		}

		break
	}

	return Parse(body, channel)
}

var pageDataRegex = regexp.MustCompile(`(?s)var\s+pageData\s*=\s*(\{.+?\});\s*`)

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	matches := pageDataRegex.FindSubmatch(data)
	if matches == nil || len(matches) < 2 {
		return nil, nil
	}

	var pageData PageData
	if err := json.Unmarshal(matches[1], &pageData); err != nil {
		return nil, nil
	}

	if pageData.Shows == nil {
		return nil, nil
	}

	programs := make([]domain.Program, 0, len(pageData.Shows))
	for _, item := range pageData.Shows {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "programetv.ro",
			Title:       item.Title,
			SubTitle:    item.TitleOriginal,
			Lang:        channel.Lang,
		}

		if item.Desc != "" {
			p.Description = item.Desc
		} else if item.Obs != "" {
			p.Description = item.Obs
		}

		if len(item.Categories) > 0 {
			p.Category = strings.Join([]string(item.Categories), ", ")
		}

		p.Season = int(item.Season)
		p.Episode = int(item.Episode)

		if item.Start != "" {
			if t, err := time.Parse(time.RFC3339, item.Start); err == nil {
				p.Start = t.UTC()
			}
		}
		if item.Stop != "" {
			if t, err := time.Parse(time.RFC3339, item.Stop); err == nil {
				p.Stop = t.UTC()
			}
		}

		if item.Icon != "" {
			p.Image = item.Icon
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type PageData struct {
	Shows []Show `json:"shows"`
}

type Show struct {
	Title         string        `json:"title"`
	TitleOriginal string        `json:"titleOriginal"`
	Desc          string        `json:"desc"`
	Obs           string        `json:"obs"`
	Categories    flexStrSlice  `json:"categories"`
	Season        flexInt       `json:"season"`
	Episode       flexInt       `json:"episode"`
	Start         string        `json:"start"`
	Stop          string        `json:"stop"`
	Icon          string        `json:"icon"`
	URL           string        `json:"url"`
	Date          string        `json:"date"`
	Rating        string        `json:"rating"`
	Credits       *Credits      `json:"credits"`
}

type Credits struct {
	Director []string `json:"director"`
	Actor    []string `json:"actor"`
}

type flexInt int

func (fi *flexInt) UnmarshalJSON(data []byte) error {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	switch v := raw.(type) {
	case float64:
		*fi = flexInt(int(v))
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			*fi = flexInt(n)
		}
	}
	return nil
}

type flexStrSlice []string

func (fs *flexStrSlice) UnmarshalJSON(data []byte) error {
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*fs = arr
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if s != "" {
			*fs = []string{s}
		}
		return nil
	}
	return nil
}
