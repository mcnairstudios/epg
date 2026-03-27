package programetvro

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

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
	return []domain.Channel{
		{ID: "programetvro-pro-tv", Name: "PRO TV", Source: "programetv.ro", SiteID: "pro-tv", Lang: "ro", XmltvID: "ProTV.ro"},
	}, nil
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
		return nil, err
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
			p.Category = strings.Join(item.Categories, ", ")
		}

		p.Season = item.Season
		p.Episode = item.Episode

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
	Title         string   `json:"title"`
	TitleOriginal string   `json:"titleOriginal"`
	Desc          string   `json:"desc"`
	Obs           string   `json:"obs"`
	Categories    []string `json:"categories"`
	Season        int      `json:"season"`
	Episode       int      `json:"episode"`
	Start         string   `json:"start"`
	Stop          string   `json:"stop"`
	Icon          string   `json:"icon"`
	URL           string   `json:"url"`
	Date          string   `json:"date"`
	Rating        string   `json:"rating"`
	Credits       *Credits `json:"credits"`
}

type Credits struct {
	Director []string `json:"director"`
	Actor    []string `json:"actor"`
}
