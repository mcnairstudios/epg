package dstv

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

func (p *provider) Name() string        { return "dstv.com" }
func (p *provider) Description() string { return "DStv African satellite television" }

const apiEndpoint = "https://www.dstv.com/umbraco/api/TvGuide"

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "dstv.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	region := strings.Split(channel.SiteID, "#")[0]
	packageParam := ""
	if region == "nga" {
		packageParam = "&package=DStv%20Premium"
	}

	url := fmt.Sprintf("%s/GetProgrammes?d=%s%s&country=%s",
		apiEndpoint, date.Format("2006-01-02"), packageParam, region)

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

	items := parseItems(body, channel)

	var programs []domain.Program
	for _, item := range items {
		details := p.loadProgramDetails(item.ID)

		prog := domain.Program{
			Channel: channel.XmltvID,
			Source:  "dstv.com",
			Title:   item.Title,
			Lang:    "en",
			Start:   parseTime(item.StartTime, channel),
			Stop:    parseTime(item.EndTime, channel),
		}

		if details != nil {
			prog.Description = details.Synopsis
			prog.Image = details.ThumbnailURI
			if len(details.SubGenres) > 0 {
				prog.Category = strings.Join(details.SubGenres, ", ")
			}
		}

		programs = append(programs, prog)
	}

	return programs, nil
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	items := parseItems(data, channel)
	var programs []domain.Program
	for _, item := range items {
		prog := domain.Program{
			Channel: channel.XmltvID,
			Source:  "dstv.com",
			Title:   item.Title,
			Lang:    "en",
			Start:   parseTime(item.StartTime, channel),
			Stop:    parseTime(item.EndTime, channel),
		}
		programs = append(programs, prog)
	}
	return programs, nil
}

func ParseWithDetails(data []byte, channel domain.Channel, detailsMap map[string]*ProgramDetails) ([]domain.Program, error) {
	items := parseItems(data, channel)
	var programs []domain.Program
	for _, item := range items {
		prog := domain.Program{
			Channel: channel.XmltvID,
			Source:  "dstv.com",
			Title:   item.Title,
			Lang:    "en",
			Start:   parseTime(item.StartTime, channel),
			Stop:    parseTime(item.EndTime, channel),
		}

		if details, ok := detailsMap[item.ID]; ok && details != nil {
			prog.Description = details.Synopsis
			prog.Image = details.ThumbnailURI
			if len(details.SubGenres) > 0 {
				prog.Category = strings.Join(details.SubGenres, ", ")
			}
		}

		programs = append(programs, prog)
	}
	return programs, nil
}

func (p *provider) loadProgramDetails(id string) *programDetails {
	url := fmt.Sprintf("%s/GetProgramme?id=%s", apiEndpoint, id)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var details programDetails
	if err := json.Unmarshal(body, &details); err != nil {
		return nil
	}
	return &details
}

var timezones = map[string]string{
	"ago": "Africa/Luanda",
	"ben": "Africa/Porto-Novo",
	"bwa": "Africa/Gaborone",
	"bfa": "Africa/Ouagadougou",
	"bdi": "Africa/Bujumbura",
	"cmr": "Africa/Douala",
	"cpv": "Atlantic/Cape_Verde",
	"tcd": "Africa/Ndjamena",
	"caf": "Africa/Bangui",
	"com": "Indian/Comoro",
	"cod": "Africa/Kinshasa",
	"dji": "Africa/Djibouti",
	"gnq": "Africa/Malabo",
	"eri": "Africa/Asmara",
	"swz": "Africa/Johannesburg",
	"eth": "Africa/Addis_Ababa",
	"gab": "Africa/Libreville",
	"gmb": "Africa/Banjul",
	"gha": "Africa/Accra",
	"gin": "Africa/Conakry",
	"gnb": "Africa/Bissau",
	"civ": "Africa/Abidjan",
	"ken": "Africa/Nairobi",
	"lbr": "Africa/Monrovia",
	"mdg": "Indian/Antananarivo",
	"mwi": "Africa/Blantyre",
	"mli": "Africa/Bamako",
	"mrt": "Africa/Nouakchott",
	"mus": "Indian/Mauritius",
	"moz": "Africa/Maputo",
	"nam": "Africa/Windhoek",
	"ner": "Africa/Niamey",
	"nga": "Africa/Lagos",
	"cog": "Africa/Brazzaville",
	"rwa": "Africa/Kigali",
	"stp": "Africa/Sao_Tome",
	"sen": "Africa/Dakar",
	"syc": "Indian/Mahe",
	"sle": "Africa/Freetown",
	"som": "Africa/Mogadishu",
	"zaf": "Africa/Johannesburg",
	"ssd": "Africa/Juba",
	"sdn": "Africa/Khartoum",
	"tza": "Africa/Dar_es_Salaam",
	"tgo": "Africa/Lome",
	"uga": "Africa/Kampala",
	"zmb": "Africa/Lusaka",
	"zwe": "Africa/Harare",
}

func parseTime(timeStr string, channel domain.Channel) time.Time {
	region := strings.Split(channel.SiteID, "#")[0]
	tzName, ok := timezones[region]
	if !ok {
		tzName = "UTC"
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		loc = time.UTC
	}
	t, err := time.ParseInLocation("2006-01-02T15:04:05", timeStr, loc)
	if err != nil {
		return time.Time{}
	}
	return t
}

func parseItems(content []byte, channel domain.Channel) []programme {
	parts := strings.SplitN(channel.SiteID, "#", 2)
	if len(parts) != 2 {
		return nil
	}
	channelID := parts[1]

	var data channelsResponse
	if err := json.Unmarshal(content, &data); err != nil {
		return nil
	}

	for _, ch := range data.Channels {
		if ch.Number == channelID {
			return ch.Programmes
		}
	}
	return nil
}

type channelsResponse struct {
	Total    int             `json:"Total"`
	Channels []channelEntry `json:"Channels"`
}

type channelEntry struct {
	Number     string      `json:"Number"`
	Name       string      `json:"Name"`
	Programmes []programme `json:"Programmes"`
}

type programme struct {
	ID        string `json:"Id"`
	StartTime string `json:"StartTime"`
	EndTime   string `json:"EndTime"`
	Title     string `json:"Title"`
}

type ProgramDetails = programDetails

type programDetails struct {
	Synopsis     string   `json:"Synopsis"`
	ThumbnailURI string   `json:"ThumbnailUri"`
	SubGenres    []string `json:"SubGenres"`
}
