package elpais

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

var madrid *time.Location

func init() {
	var err error
	madrid, err = time.LoadLocation("Europe/Madrid")
	if err != nil {
		panic(err)
	}

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

func (p *provider) Name() string        { return "programacion-tv.elpais.com" }
func (p *provider) Description() string { return "El Pais TV guide (Spain, 195 channels)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "programacion-tv.elpais.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	gridURL := fmt.Sprintf(
		"https://programacion-tv.elpais.com/data/parrilla_%s.json",
		date.In(madrid).Format("02012006"),
	)

	gridData, err := p.get(gridURL)
	if err != nil {
		return nil, err
	}

	programsURL := fmt.Sprintf(
		"https://programacion-tv.elpais.com/data/programas/%s.json",
		channel.SiteID,
	)

	programsData, err := p.get(programsURL)
	if err != nil {
		return nil, err
	}

	return Parse(gridData, programsData, channel)
}

func (p *provider) get(u string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from elpais", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func Parse(gridData, programsData []byte, channel domain.Channel) ([]domain.Program, error) {
	items, err := parseItems(gridData, channel.SiteID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}

	details, err := parseProgramDetails(programsData)
	if err != nil {
		details = nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		start, err := parseTime(item.IniDate)
		if err != nil {
			continue
		}
		stop, err := parseTime(item.EndDate)
		if err != nil {
			continue
		}

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "programacion-tv.elpais.com",
			Title:   item.Title,
			Lang:    channel.Lang,
			Start:   start,
			Stop:    stop,
		}

		if d, ok := details[item.IDPrograma]; ok {
			if d.EpisodeTitle != "" {
				p.SubTitle = d.EpisodeTitle
			}
			if d.EpisodeDescription != "" {
				p.Description = d.EpisodeDescription
			} else if item.Description != "" {
				p.Description = item.Description
			}
			p.Category = parseCategory(d.TxtGenre, d.TxtSubgenre)
			p.Image = parseImage(d.Image)
			p.Season = parseNumber(d.Season)
			p.Episode = parseNumber(d.Episode)
		} else {
			p.Description = item.Description
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func parseItems(data []byte, siteID string) ([]gridProgram, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var grid []gridChannel
	if err := json.Unmarshal(data, &grid); err != nil {
		return nil, err
	}

	for _, ch := range grid {
		if ch.IDCanal == siteID {
			return ch.Programas, nil
		}
	}

	return nil, nil
}

func parseProgramDetails(data []byte) (map[string]programDetail, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var details []programDetail
	if err := json.Unmarshal(data, &details); err != nil {
		return nil, err
	}

	m := make(map[string]programDetail, len(details))
	for _, d := range details {
		m[d.IDPrograma] = d
	}
	return m, nil
}

func parseTime(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", s, madrid)
}

func parseNumber(s string) int {
	if s == "" {
		return 0
	}
	n, _ := strconv.Atoi(s)
	return n
}

func parseCategory(genre, subgenre string) string {
	parts := make([]string, 0, 2)
	if genre != "" {
		parts = append(parts, genre)
	}
	if subgenre != "" {
		parts = append(parts, subgenre)
	}
	return strings.Join(parts, "/")
}

func parseImage(path string) string {
	if path == "" {
		return ""
	}
	u, err := url.Parse(path)
	if err != nil {
		return ""
	}
	base, _ := url.Parse("https://programacion-tv.elpais.com/")
	return base.ResolveReference(u).String()
}

type gridChannel struct {
	IDCanal   string        `json:"idCanal"`
	Programas []gridProgram `json:"programas"`
}

type gridProgram struct {
	ID          string `json:"id"`
	IDCanal     string `json:"idCanal"`
	IniDate     string `json:"iniDate"`
	EndDate     string `json:"endDate"`
	Title       string `json:"title"`
	Description string `json:"description"`
	IDPrograma  string `json:"id_programa"`
}

type programDetail struct {
	IDPrograma         string `json:"id_programa"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	TxtGenre           string `json:"txt_genre"`
	TxtSubgenre        string `json:"txt_subgenre"`
	Season             string `json:"season"`
	Episode            string `json:"episode"`
	EpisodeTitle       string `json:"episode_title"`
	EpisodeDescription string `json:"episode_description"`
	Director           string `json:"director"`
	Script             string `json:"script"`
	Actors             string `json:"actors"`
	Producer           string `json:"producer"`
	PresentedBy        string `json:"presented_by"`
	Music              string `json:"music"`
	GuestActors        string `json:"guest_actors"`
	Image              string `json:"image"`
}
