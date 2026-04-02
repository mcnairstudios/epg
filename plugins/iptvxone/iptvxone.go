package iptvxone

import (
	"compress/gzip"
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

func (p *provider) Name() string        { return "epg.iptvx.one" }
func (p *provider) Description() string { return "IPTVX.one Russian IPTV EPG source" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "epg.iptvx.one")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	req, err := http.NewRequest(http.MethodGet, "https://iptvx.one/epg/epg_noarch.xml.gz", nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from iptvxone", resp.StatusCode)
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress gzip: %w", err)
	}
	defer gr.Close()

	body, err := io.ReadAll(gr)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	var tv tvDoc
	if err := xml.Unmarshal(data, &tv); err != nil {
		return nil, fmt.Errorf("failed to parse XMLTV: %w", err)
	}

	day := date.Truncate(24 * time.Hour).UTC()
	nextDay := day.Add(24 * time.Hour)

	var programs []domain.Program
	for _, prog := range tv.Programmes {
		if prog.Channel != channel.SiteID {
			continue
		}

		start, err := parseXMLTVTime(prog.Start)
		if err != nil {
			continue
		}
		startUTC := start.UTC()

		if startUTC.Before(day) || !startUTC.Before(nextDay) {
			continue
		}

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "epg.iptvx.one",
			Lang:    channel.Lang,
			Start:   startUTC,
		}

		if stop, err := parseXMLTVTime(prog.Stop); err == nil {
			p.Stop = stop.UTC()
		}

		if len(prog.Title) > 0 {
			p.Title = prog.Title[0].Value
		}
		if len(prog.Desc) > 0 {
			p.Description = prog.Desc[0].Value
		}

		programs = append(programs, p)
	}

	if programs == nil {
		programs = []domain.Program{}
	}

	return programs, nil
}

func ParseGzipped(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	gr, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return []domain.Program{}, nil
	}
	defer gr.Close()

	body, err := io.ReadAll(gr)
	if err != nil {
		return []domain.Program{}, nil
	}

	return Parse(body, channel, date)
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
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}

type tvDoc struct {
	XMLName    xml.Name    `xml:"tv"`
	Programmes []programme `xml:"programme"`
}

type programme struct {
	Start   string    `xml:"start,attr"`
	Stop    string    `xml:"stop,attr"`
	Channel string    `xml:"channel,attr"`
	Title   []xmlText `xml:"title"`
	Desc    []xmlText `xml:"desc"`
}

type xmlText struct {
	Lang  string `xml:"lang,attr,omitempty"`
	Value string `xml:",chardata"`
}

func (p *provider) Timezone() string { return "Europe/Moscow" }
