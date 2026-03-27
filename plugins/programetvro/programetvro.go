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

	url := fmt.Sprintf(
		"https://www.programetv.ro/program-tv/%s/%s/",
		channel.SiteID,
		dayName,
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
		return nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

var pageDataRegex = regexp.MustCompile(`var pageData = ({.+?});`)

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

// flexInt handles JSON values that may be a number, a string, or null.
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

// flexStrSlice handles JSON values that may be an array of strings, a single string, or null.
type flexStrSlice []string

func (fs *flexStrSlice) UnmarshalJSON(data []byte) error {
	// Try array first
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*fs = arr
		return nil
	}
	// Try single string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if s != "" {
			*fs = []string{s}
		}
		return nil
	}
	// Ignore unrecognized types
	return nil
}
