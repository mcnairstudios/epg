package directvar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

var buenosAires *time.Location

func init() {
	var err error
	buenosAires, err = time.LoadLocation("America/Argentina/Buenos_Aires")
	if err != nil {
		panic(err)
	}
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "directv.com.ar" }
func (p *provider) Description() string { return "DirecTV Argentina electronic program guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "directvar-207", Name: "A&E HD", Source: "directv.com.ar", SiteID: "207#A&amp;EHD", Lang: "es", XmltvID: "AEHDSouth.us"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	channelNum, channelName := parseSiteID(channel.SiteID)

	d := date.In(buenosAires)
	body := requestBody{
		FilterParameters: filterParameters{
			Day:                  d.Day(),
			Time:                 0,
			Minute:               0,
			Month:                int(d.Month()),
			Year:                 d.Year(),
			OffSetValue:          0,
			HomeScreenFilter:     "",
			FiltersScreenFilters: []string{""},
			IsHD:                 "",
			IsChannelDetails:     "Y",
			ChannelNum:           channelNum,
			ChannelName:          channelName,
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, "https://www.directv.com.ar/guia/ChannelDetail.aspx/GetProgramming", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Cookie", "PGCSS=16; PGLang=S; PGCulture=es-AR;")
	req.Header.Set("Origin", "https://www.directv.com.ar")
	req.Header.Set("Referer", fmt.Sprintf("https://www.directv.com.ar/guia/ChannelDetail.aspx?id=%s&name=%s", channelNum, channelName))
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")

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
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	channelNum, channelName := parseSiteID(channel.SiteID)

	var programList []program
	for _, ch := range response.D {
		if fmt.Sprintf("%d", ch.ChannelNumber) == channelNum && ch.ChannelName == channelName {
			programList = ch.ProgramList
			break
		}
	}

	programs := make([]domain.Program, 0, len(programList))
	for _, item := range programList {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "directv.com.ar",
			Title:       item.Title,
			Description: item.Description,
			Lang:        channel.Lang,
		}

		if item.StartTimeString != "" {
			if t, err := time.ParseInLocation("1/2/2006 3:04:05 PM", item.StartTimeString, buenosAires); err == nil {
				p.Start = t.UTC()
			}
		}
		if item.EndTimeString != "" {
			if t, err := time.ParseInLocation("1/2/2006 3:04:05 PM", item.EndTimeString, buenosAires); err == nil {
				p.Stop = t.UTC()
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func parseSiteID(siteID string) (string, string) {
	parts := strings.SplitN(siteID, "#", 2)
	if len(parts) != 2 {
		return siteID, ""
	}
	channelName := strings.ReplaceAll(parts[1], "&amp;", "&")
	return parts[0], channelName
}

type requestBody struct {
	FilterParameters filterParameters `json:"filterParameters"`
}

type filterParameters struct {
	Day                  int      `json:"day"`
	Time                 int      `json:"time"`
	Minute               int      `json:"minute"`
	Month                int      `json:"month"`
	Year                 int      `json:"year"`
	OffSetValue          int      `json:"offSetValue"`
	HomeScreenFilter     string   `json:"homeScreenFilter"`
	FiltersScreenFilters []string `json:"filtersScreenFilters"`
	IsHD                 string   `json:"isHd"`
	IsChannelDetails     string   `json:"isChannelDetails"`
	ChannelNum           string   `json:"channelNum"`
	ChannelName          string   `json:"channelName"`
}

type apiResponse struct {
	D []channelData `json:"d"`
}

type channelData struct {
	ChannelName     string    `json:"ChannelName"`
	ChannelFullName string    `json:"ChannelFullName"`
	ChannelNumber   int       `json:"ChannelNumber"`
	ProgramList     []program `json:"ProgramList"`
}

type program struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	Rating          string `json:"rating"`
	StartTimeString string `json:"startTimeString"`
	EndTimeString   string `json:"endTimeString"`
	ImageURL        string `json:"imageUrl"`
}
