package dishtvIn

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "dishtv.in" }
func (p *provider) Description() string { return "DishTV India TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "dishtv-142639", Name: "&prive HD", Source: "dishtv.in", SiteID: "142639", Lang: "en", XmltvID: "AndpriveHD.in"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	body := `{"allowPastEvents":true,"channelid":"` + channel.SiteID + `","date":"` + date.Format("02/01/2006") + `"}`

	req, err := http.NewRequest(http.MethodPost, "https://epg.mysmartstick.com/dishtv/api/v1/epg/entities/programs", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

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

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var items []programItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, nil
	}

	loc, _ := time.LoadLocation("Asia/Kolkata")
	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		prog := domain.Program{
			Channel: channel.XmltvID,
			Source:  "dishtv.in",
			Lang:    channel.Lang,
		}

		if item.Start != "" {
			cleaned := strings.TrimSuffix(item.Start, "Z")
			if t, err := time.ParseInLocation("2006-01-02T15:04:05", cleaned, loc); err == nil {
				prog.Start = t
			}
		}
		if item.Stop != "" {
			cleaned := strings.TrimSuffix(item.Stop, "Z")
			if t, err := time.ParseInLocation("2006-01-02T15:04:05", cleaned, loc); err == nil {
				prog.Stop = t
			}
		}

		if len(item.Regional) > 0 {
			for _, r := range item.Regional {
				if r.Title != "" {
					prog.Title = r.Title
					break
				}
			}
		}

		if item.EpisodeNum != "" {
			if n, err := strconv.Atoi(item.EpisodeNum); err == nil {
				prog.Episode = n
			}
		}

		if item.Images.Landscape.Size1280x720 != "" {
			prog.Image = item.Images.Landscape.Size1280x720
		}

		programs = append(programs, prog)
	}

	return programs, nil
}

type programItem struct {
	Start          string              `json:"start"`
	Stop           string              `json:"stop"`
	Regional       map[string]regional `json:"regional"`
	Credits        credits             `json:"credits"`
	EpisodeNum     string              `json:"episode-num"`
	ProductionYear string              `json:"productionyear"`
	ProgrammeURL   string              `json:"programmeurl"`
	Images         images              `json:"images"`
}

type regional struct {
	LanguageCode string `json:"languagecode"`
	Title        string `json:"title"`
	Desc         string `json:"desc"`
	Genre        string `json:"genre"`
}

type credits struct {
	Actors    []string `json:"actors"`
	Directors []string `json:"directors"`
	Producers []string `json:"producers"`
}

type images struct {
	Landscape landscapeImages `json:"landscape"`
}

type landscapeImages struct {
	Size880x660  string `json:"880x660"`
	Size1280x720 string `json:"1280x720"`
}
