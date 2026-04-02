package nzxmltv

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

const apiEndpoint = "https://nzxmltv.com"

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{client: httpclient.Browser()})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "nzxmltv.com" }
func (p *provider) Description() string { return "New Zealand XMLTV guide (Freeview/Sky/RedBull/Pluto)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "nzxmltv.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	path, channelID, ok := splitSiteID(channel.SiteID)
	if !ok {
		return nil, fmt.Errorf("invalid site_id format: %s", channel.SiteID)
	}

	url := fmt.Sprintf("%s/%s.xml", apiEndpoint, path)
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
		return nil, fmt.Errorf("HTTP %d from nzxmltv", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, channelID, date)
}

func Parse(data []byte, channel domain.Channel, channelID string, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var tv tvDoc
	if err := xml.Unmarshal(data, &tv); err != nil {
		return nil, fmt.Errorf("failed to parse XMLTV: %w", err)
	}

	day := date.Truncate(24 * time.Hour)
	nextDay := day.Add(24 * time.Hour)

	var programs []domain.Program
	for _, prog := range tv.Programmes {
		if prog.Channel != channelID {
			continue
		}

		start, err := parseXMLTVTime(prog.Start)
		if err != nil {
			continue
		}
		if start.Before(day) || !start.Before(nextDay) {
			continue
		}

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "nzxmltv.com",
			Lang:    channel.Lang,
			Start:   start,
		}

		if t, err := parseXMLTVTime(prog.Stop); err == nil {
			p.Stop = t
		}

		if len(prog.Title) > 0 {
			p.Title = prog.Title[0].Value
		}
		if len(prog.Desc) > 0 {
			p.Description = prog.Desc[0].Value
		}
		if prog.Icon != nil {
			p.Image = prog.Icon.Src
		}

		for _, ep := range prog.EpisodeNums {
			if ep.System == "xmltv_ns" {
				p.Season, p.Episode = parseXmltvNS(ep.Value)
				break
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type tvDoc struct {
	XMLName    xml.Name    `xml:"tv"`
	Programmes []programme `xml:"programme"`
}

type programme struct {
	Start       string       `xml:"start,attr"`
	Stop        string       `xml:"stop,attr"`
	Channel     string       `xml:"channel,attr"`
	Title       []xmlName    `xml:"title"`
	Desc        []xmlName    `xml:"desc"`
	Icon        *xmlIcon     `xml:"icon"`
	EpisodeNums []episodeNum `xml:"episode-num"`
}

type xmlName struct {
	Lang  string `xml:"lang,attr,omitempty"`
	Value string `xml:",chardata"`
}

type xmlIcon struct {
	Src string `xml:"src,attr"`
}

type episodeNum struct {
	System string `xml:"system,attr"`
	Value  string `xml:",chardata"`
}

func parseXMLTVTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	formats := []string{
		"20060102150405 -0700",
		"20060102150405 +0000",
		"20060102150405",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}

func parseXmltvNS(value string) (int, int) {
	parts := strings.Split(value, ".")
	if len(parts) < 2 {
		return 0, 0
	}
	s, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	e, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	return s + 1, e + 1
}

func splitSiteID(siteID string) (string, string, bool) {
	parts := strings.SplitN(siteID, "#", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func (p *provider) Timezone() string { return "Pacific/Auckland" }
