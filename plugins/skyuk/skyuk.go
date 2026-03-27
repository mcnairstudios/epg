package skyuk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
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

func (p *provider) Name() string        { return "sky.com" }
func (p *provider) Description() string { return "Sky UK electronic programme guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "sky.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	day := date.Truncate(24 * time.Hour).UTC()

	url := fmt.Sprintf(
		"https://awk.epgsky.com/hawk/linear/schedule/%s/%s",
		day.Format("20060102"),
		channel.SiteID,
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

	return Parse(body, channel, day)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	var programs []domain.Program
	for _, sched := range response.Schedule {
		if sched.SID != channel.SiteID {
			continue
		}

		sort.Slice(sched.Events, func(i, j int) bool {
			return sched.Events[i].St < sched.Events[j].St
		})

		for _, event := range sched.Events {
			start := time.Unix(event.St, 0).UTC()
			y1, m1, d1 := start.Date()
			y2, m2, d2 := date.Date()
			if y1 != y2 || m1 != m2 || d1 != d2 {
				continue
			}

			stop := start.Add(time.Duration(event.D) * time.Second)
			image := fmt.Sprintf("https://images.metadata.sky.com/pd-image/%s/16-9/640", event.ProgrammeUUID)

			p := domain.Program{
				Channel:     channel.XmltvID,
				Source:      "sky.com",
				Title:       event.T,
				Description: event.Sy,
				Season:      event.SeasonNumber,
				Episode:     event.EpisodeNumber,
				Image:       image,
				Lang:        channel.Lang,
				Start:       start,
				Stop:        stop,
			}

			programs = append(programs, p)
		}
	}

	if programs == nil {
		programs = []domain.Program{}
	}

	return programs, nil
}

type apiResponse struct {
	Schedule []schedule `json:"schedule"`
}

type schedule struct {
	SID    string  `json:"sid"`
	Events []event `json:"events"`
}

type event struct {
	St             int64  `json:"st"`
	D              int64  `json:"d"`
	T              string `json:"t"`
	Sy             string `json:"sy"`
	ProgrammeUUID  string `json:"programmeuuid"`
	SeasonNumber   int    `json:"seasonnumber"`
	EpisodeNumber  int    `json:"episodenumber"`
}
