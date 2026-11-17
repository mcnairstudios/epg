package epgshare

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/fetchcache"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{
		client: httpclient.Browser(),
		cache:  newCache(),
	})
}

type provider struct {
	client httpClient
	cache  *fetchcache.Cache[index]
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "epgshare01.online" }
func (p *provider) Description() string { return "EPGShare01 XMLTV aggregator (20706 channels)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	chs, err := channels.Load(channelsData, "epgshare01.online")
	if err != nil {
		return nil, err
	}
	return channels.GroupBy(chs, tagOf), nil
}

func tagOf(ch domain.Channel) string {
	tag, _, _ := strings.Cut(ch.SiteID, "#")
	return tag
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	idx, err := p.cache.Get(tagOf(channel), func() (index, error) { return p.download(tagOf(channel)) })
	if err != nil {
		return nil, err
	}
	return selectPrograms(idx, channel, date), nil
}

func (p *provider) download(tag string) (index, error) {
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
		return nil, &fetchcache.StatusError{Code: resp.StatusCode, Source: "epgshare"}
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	return buildIndex(gz)
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}
	idx, err := buildIndex(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return selectPrograms(idx, channel, date), nil
}

type entry struct {
	day         string
	start, stop time.Time
	title, desc string
}

type index map[string][]entry

func buildIndex(r io.Reader) (index, error) {
	dec := xml.NewDecoder(r)
	idx := make(index)
	sawElement := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		sawElement = true
		if se.Name.Local != "programme" {
			continue
		}
		var prog xmlProgramme
		if err := dec.DecodeElement(&prog, &se); err != nil {
			return nil, err
		}
		raw := strings.TrimSpace(prog.Start)
		start := parseXMLTVTime(prog.Start)
		if start.IsZero() {
			continue
		}
		e := entry{day: raw[:8], start: start, stop: parseXMLTVTime(prog.Stop)}
		if len(prog.Title) > 0 {
			e.title = prog.Title[0].Value
		}
		if len(prog.Desc) > 0 {
			e.desc = prog.Desc[0].Value
		}
		idx[prog.Channel] = append(idx[prog.Channel], e)
	}
	if !sawElement {
		return nil, errors.New("epgshare: no XML elements found")
	}
	return idx, nil
}

func selectPrograms(idx index, channel domain.Channel, date time.Time) []domain.Program {
	_, channelID, _ := strings.Cut(channel.SiteID, "#")
	day := date.Format("20060102")

	programs := []domain.Program{}
	for _, e := range idx[channelID] {
		if e.day != day {
			continue
		}
		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "epgshare01.online",
			Lang:        channel.Lang,
			Start:       e.start,
			Stop:        e.stop,
			Title:       e.title,
			Description: e.desc,
		})
	}
	return programs
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
