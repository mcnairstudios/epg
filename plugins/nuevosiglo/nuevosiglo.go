package nuevosiglo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
	"golang.org/x/net/html"
)

//go:embed channels.json
var channelsData []byte

var montevideo *time.Location

func init() {
	montevideo = time.FixedZone("UYT", -3*60*60)
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

func (p *provider) Name() string        { return "nuevosiglo.com.uy" }
func (p *provider) Description() string { return "Nuevo Siglo Uruguay (173 channels)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "nuevosiglo.com.uy")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf("https://www.nuevosiglo.com.uy/programacion/getGrilla?fecha=%s", date.Format("2006/01/02"))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	scheduleIDs := parseScheduleIDs(data, channel)
	if len(scheduleIDs) == 0 {
		return []domain.Program{}, nil
	}

	var programs []domain.Program
	for _, id := range scheduleIDs {
		detail, err := p.fetchProgramDetail(id)
		if err != nil || detail == nil {
			continue
		}
		prog := parseProgramDetail(detail, channel)
		programs = append(programs, prog)
	}

	return programs, nil
}

func (p *provider) fetchProgramDetail(id string) (*programDetail, error) {
	url := fmt.Sprintf("https://www.nuevosiglo.com.uy/Programacion/getScheduleXId/%s", id)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return ParseProgramDetail(data)
}

func ParseProgramDetail(data []byte) (*programDetail, error) {
	var detail programDetail
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

func parseProgramDetail(detail *programDetail, channel domain.Channel) domain.Program {
	p := domain.Program{
		Channel:     channel.XmltvID,
		Source:      "nuevosiglo.com.uy",
		Title:       detail.MainTitle,
		Description: detail.ShortArgument,
		Lang:        channel.Lang,
	}

	if detail.Image != "" {
		p.Image = fmt.Sprintf("https://img-ns.s3.amazonaws.com/grid_data/%s", detail.Image)
	}

	if detail.TimeStart != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", detail.TimeStart, montevideo); err == nil {
			p.Start = t.UTC()
		}
	}
	if detail.TimeEnd != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", detail.TimeEnd, montevideo); err == nil {
			p.Stop = t.UTC()
		}
	}

	return p
}

func parseScheduleIDs(data []byte, channel domain.Channel) []string {
	if len(data) == 0 {
		return nil
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil
	}

	var ids []string
	var foundChannel bool

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			alt := getAttr(n, "alt")
			if alt == channel.SiteID {
				foundChannel = true
			} else if foundChannel && alt != "" {
				foundChannel = false
				return
			}
		}

		if foundChannel && n.Type == html.ElementNode && n.Data == "div" {
			schedule := getAttr(n, "data-schedule")
			if schedule != "" {
				ids = append(ids, schedule)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return ids
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

type programDetail struct {
	MainTitle      string `json:"main_title"`
	ShortArgument  string `json:"short_argument"`
	TimeStart      string `json:"time_start"`
	TimeEnd        string `json:"time_end"`
	Image          string `json:"image"`
	Year           string `json:"year"`
	ParentalRating string `json:"parental_rating"`
	Actors         string `json:"actors"`
}
