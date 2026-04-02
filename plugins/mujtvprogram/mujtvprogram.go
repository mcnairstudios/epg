package mujtvprogram

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

var pragueTZ *time.Location

//go:embed channels.json
var channelsData []byte

func init() {
	var err error
	pragueTZ, err = time.LoadLocation("Europe/Prague")
	if err != nil {
		pragueTZ = time.FixedZone("CET", 1*60*60)
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

func (p *provider) Name() string        { return "mujtvprogram.cz" }
func (p *provider) Description() string { return "Czech TV program guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "mujtvprogram.cz")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	d := date.Truncate(24 * time.Hour).UTC()
	diff := int(d.Sub(now).Hours() / 24)

	url := fmt.Sprintf(
		"https://services.mujtvprogram.cz/tvprogram2services/services/tvprogrammelist_mobile.php?channel_cid=%s&day=%d",
		channel.SiteID,
		diff,
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
		return nil, fmt.Errorf("HTTP %d from mujtvprogram", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var guide tvProgramProgrammes
	if err := xml.Unmarshal(data, &guide); err != nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(guide.Programmes))
	for _, item := range guide.Programmes {
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "mujtvprogram.cz",
			Title:   item.Name,
			Lang:    channel.Lang,
		}

		if item.LongDescription != "" {
			p.Description = item.LongDescription
		} else if item.ShortDescription != "" {
			p.Description = item.ShortDescription
		}

		if item.ProgrammeType != nil && item.ProgrammeType.Name != "" {
			p.Category = item.ProgrammeType.Name
		}

		if item.StartDate != "" {
			if t, err := time.ParseInLocation("02.01.2006 15:04", item.StartDate, pragueTZ); err == nil {
				p.Start = t.UTC()
			}
		}

		if item.EndDate != "" {
			if t, err := time.ParseInLocation("02.01.2006 15:04", item.EndDate, pragueTZ); err == nil {
				p.Stop = t.UTC()
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func parseList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ", ")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

type tvProgramProgrammes struct {
	XMLName    xml.Name    `xml:"tv-program-programmes"`
	Programmes []programme `xml:"programme"`
}

type programme struct {
	Name             string         `xml:"name"`
	ShortDescription string         `xml:"shortDescription"`
	LongDescription  string         `xml:"longDescription"`
	Year             string         `xml:"year"`
	Directors        string         `xml:"directors"`
	Actors           string         `xml:"actors"`
	StartDate        string         `xml:"startDate"`
	EndDate          string         `xml:"endDate"`
	ProgrammeType    *programmeType `xml:"programme-type"`
}

type programmeType struct {
	Name string `xml:"name"`
}

func (p *provider) Timezone() string { return "Europe/Prague" }
