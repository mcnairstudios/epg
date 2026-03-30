package energeek

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

func (p *provider) Name() string        { return "energeek.cl" }
func (p *provider) Description() string { return "EnerGeek Chilean TV listings" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "energeek.cl")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := "https://raw.githubusercontent.com/luisms123/tdt/master/guiaenergeek.xml"

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
		return nil, fmt.Errorf("HTTP %d from energeek", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	var tv xmlTV
	if err := xml.Unmarshal(data, &tv); err != nil {
		return []domain.Program{}, nil
	}

	dateStr := date.Format("20060102")
	var programs []domain.Program

	for _, prog := range tv.Programmes {
		if prog.Channel != channel.SiteID {
			continue
		}

		start := parseXMLTVTime(prog.Start)
		if start.IsZero() {
			continue
		}

		if start.UTC().Format("20060102") != dateStr {
			continue
		}

		stop := parseXMLTVTime(prog.Stop)

		title := ""
		if len(prog.Title) > 0 {
			title = prog.Title[0].Value
		}

		desc := ""
		if len(prog.Desc) > 0 {
			desc = prog.Desc[0].Value
		}

		image := ""
		if len(prog.Icon) > 0 {
			image = prog.Icon[0].Src
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "energeek.cl",
			Title:       title,
			Description: desc,
			Image:       image,
			Lang:        channel.Lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		})
	}

	if programs == nil {
		return []domain.Program{}, nil
	}
	return programs, nil
}

func parseXMLTVTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}

	layouts := []string{
		"20060102150405 -0700",
		"20060102150405",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

type xmlTV struct {
	XMLName    xml.Name     `xml:"tv"`
	Programmes []xmlProgram `xml:"programme"`
}

type xmlProgram struct {
	Start   string    `xml:"start,attr"`
	Stop    string    `xml:"stop,attr"`
	Channel string    `xml:"channel,attr"`
	Title   []xmlText `xml:"title"`
	Desc    []xmlText `xml:"desc"`
	Icon    []xmlIcon `xml:"icon"`
}

type xmlText struct {
	Lang  string `xml:"lang,attr"`
	Value string `xml:",chardata"`
}

type xmlIcon struct {
	Src string `xml:"src,attr"`
}
