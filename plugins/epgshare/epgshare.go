package epgshare

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

func (p *provider) Name() string        { return "epgshare01.online" }
func (p *provider) Description() string { return "EPGShare01 XMLTV aggregator (20706 channels)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "epgshare01.online")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	tag, _, _ := strings.Cut(channel.SiteID, "#")

	url := fmt.Sprintf("https://epgshare01.online/epgshare01/epg_ripper_%s.xml.gz", tag)

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
		return nil, fmt.Errorf("HTTP %d from epgshare", resp.StatusCode)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	data, err := io.ReadAll(gz)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel, date)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	_, channelID, _ := strings.Cut(channel.SiteID, "#")

	var tv xmlTV
	if err := xml.Unmarshal(data, &tv); err != nil {
		return nil, err
	}

	datePrefix := date.Format("20060102")

	var programs []domain.Program
	for _, prog := range tv.Programmes {
		if prog.Channel != channelID {
			continue
		}

		if !strings.HasPrefix(strings.TrimSpace(prog.Start), datePrefix) {
			continue
		}

		start := parseXMLTVTime(prog.Start)
		stop := parseXMLTVTime(prog.Stop)

		if start.IsZero() {
			continue
		}

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "epgshare01.online",
			Lang:    channel.Lang,
			Start:   start,
			Stop:    stop,
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
		return []domain.Program{}, nil
	}

	return programs, nil
}

func parseXMLTVTime(s string) time.Time {
	raw := strings.TrimSpace(s)
	if len(raw) < 14 {
		return time.Time{}
	}

	if len(raw) > 14 {
		t, err := time.Parse("20060102150405 -0700", raw)
		if err == nil {
			return t.UTC()
		}
	}

	t, err := time.Parse("20060102150405", raw[:14])
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

type xmlTV struct {
	Programmes []xmlProgramme `xml:"programme"`
}

type xmlProgramme struct {
	Start   string    `xml:"start,attr"`
	Stop    string    `xml:"stop,attr"`
	Channel string    `xml:"channel,attr"`
	Title   []xmlText `xml:"title"`
	Desc    []xmlText `xml:"desc"`
}

type xmlText struct {
	Lang  string `xml:"lang,attr"`
	Value string `xml:",chardata"`
}

func (p *provider) Timezone() string { return "America/New_York" }
