package canalplus

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func (p *provider) Name() string        { return "canalplus.com" }
func (p *provider) Description() string { return "Canal+ TV schedule" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "canalplus.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour).UTC()

	url := fmt.Sprintf(
		"https://hodor.canalplus.pro/api/v2/mycanal/channels/token/%s/broadcasts/day/%d",
		channel.SiteID,
		int(start.Sub(time.Now().UTC().Truncate(24*time.Hour)).Hours()/24),
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
		return nil, fmt.Errorf("HTTP %d from canalplus", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response ScheduleResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	if response.TimeSlices == nil {
		return nil, nil
	}

	var items []ScheduleItem
	for _, ts := range response.TimeSlices {
		items = append(items, ts.Contents...)
	}

	programs := make([]domain.Program, 0, len(items))
	for i, item := range items {
		start := time.Unix(0, item.StartTime*int64(time.Millisecond)).UTC()
		stop := start.Add(1 * time.Hour)
		if i+1 < len(items) {
			stop = time.Unix(0, items[i+1].StartTime*int64(time.Millisecond)).UTC()
		}

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "canalplus.com",
			Title:   item.Title,
			Lang:    channel.Lang,
			Start:   start,
			Stop:    stop,
		})
	}

	return programs, nil
}

func ParseDetail(data []byte) (*DetailInfo, error) {
	var response DetailResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	if response.Detail == nil || response.Detail.Informations == nil {
		return nil, nil
	}

	info := response.Detail.Informations
	result := &DetailInfo{
		Description:    info.Summary,
		Image:          info.URLImage,
		ProductionYear: info.ProductionYear,
	}

	for _, p := range info.Personnalities {
		names := make([]string, 0, len(p.PersonnalitiesList))
		for _, person := range p.PersonnalitiesList {
			names = append(names, person.Title)
		}
		switch p.Prefix {
		case "Avec :":
			result.Actors = names
		case "De :":
			result.Directors = names
		case "Scénario :":
			result.Writers = names
		case "Musique :":
			result.Composers = names
		case "Présenté par :":
			result.Presenters = names
		}
	}

	if info.ParentalRatings != nil {
		for _, r := range info.ParentalRatings {
			if r.Authority == "CSA" {
				value := r.Value
				if value == "1" {
					continue
				}
				switch value {
				case "2":
					value = "-10"
				case "3":
					value = "-12"
				case "4":
					value = "-16"
				case "5":
					value = "-18"
				}
				result.Rating = &Rating{System: r.Authority, Value: value}
			}
		}
	}

	return result, nil
}

type ScheduleResponse struct {
	TimeSlices []TimeSlice `json:"timeSlices"`
}

type TimeSlice struct {
	TimeSlice string         `json:"timeSlice"`
	Contents  []ScheduleItem `json:"contents"`
}

type ScheduleItem struct {
	ContentID string    `json:"contentID"`
	Title     string    `json:"title"`
	Subtitle  string    `json:"subtitle"`
	StartTime int64     `json:"startTime"`
	OnClick   *OnClick  `json:"onClick"`
}

type OnClick struct {
	URLPage string `json:"URLPage"`
}

type DetailResponse struct {
	Detail *Detail `json:"detail"`
}

type Detail struct {
	Informations *Informations `json:"informations"`
}

type Informations struct {
	Title           string           `json:"title"`
	URLImage        string           `json:"URLImage"`
	Summary         string           `json:"summary"`
	ProductionYear  string           `json:"productionYear"`
	Personnalities  []Personnality   `json:"personnalities"`
	ParentalRatings []ParentalRating `json:"parentalRatings"`
}

type Personnality struct {
	Prefix             string   `json:"prefix"`
	PersonnalitiesList []Person `json:"personnalitiesList"`
}

type Person struct {
	Title string `json:"title"`
}

type ParentalRating struct {
	Value     string `json:"value"`
	Authority string `json:"authority"`
}

type DetailInfo struct {
	Description    string
	Image          string
	ProductionYear string
	Actors         []string
	Directors      []string
	Writers        []string
	Composers      []string
	Presenters     []string
	Rating         *Rating
}

type Rating struct {
	System string
	Value  string
}
