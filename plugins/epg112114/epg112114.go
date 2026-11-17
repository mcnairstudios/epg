package epg112114

import (
	"encoding/xml"
	"io"
	"net/http"
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
	plugin.Register(&provider{client: httpclient.Browser(), docs: newDocs()})
}

type provider struct {
	client httpClient
	docs   *fetchcache.Cache[doc]
}

type doc map[string][]programme

func newDocs() *fetchcache.Cache[doc] {
	return fetchcache.New(fetchcache.Options[doc]{
		TTL:        fetchcache.DocTTL[doc](15 * time.Minute),
		MaxEntries: 1,
	})
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "epg.112114.xyz" }
func (p *provider) Description() string { return "Chinese IPTV EPG from 112114.xyz" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "epg.112114.xyz")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d, err := p.docs.Get("pp.xml", p.download)
	if err != nil {
		return nil, err
	}
	return selectPrograms(d, channel, date), nil
}

func (p *provider) download() (doc, error) {
	url := "https://epg.112114.xyz/pp.xml"

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
		return nil, &fetchcache.StatusError{Code: resp.StatusCode, Source: "epg112114"}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	d, err := buildDoc(body)
	if err != nil {
		return doc{}, nil
	}
	return d, nil
}

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d, err := buildDoc(data)
	if err != nil {
		return []domain.Program{}, nil
	}
	return selectPrograms(d, channel, date), nil
}

func buildDoc(data []byte) (doc, error) {
	d := make(doc)
	if len(data) == 0 {
		return d, nil
	}
	var tv xmlTV
	if err := xml.Unmarshal(data, &tv); err != nil {
		return nil, err
	}
	for _, prog := range tv.Programmes {
		d[prog.Channel] = append(d[prog.Channel], prog)
	}
	return d, nil
}

func selectPrograms(d doc, channel domain.Channel, date time.Time) []domain.Program {
	dateStr := date.Format("20060102")
	var programs []domain.Program

	for _, prog := range d[channel.SiteID] {
		start, err := parseXMLTVTime(prog.Start)
		if err != nil {
			continue
		}

		if start.UTC().Format("20060102") != dateStr {
			continue
		}

		stop, err := parseXMLTVTime(prog.Stop)
		if err != nil {
			continue
		}

		var title string
		if len(prog.Titles) > 0 {
			title = prog.Titles[0].Value
		}

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "epg.112114.xyz",
			Title:   title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		})
	}

	if programs == nil {
		programs = []domain.Program{}
	}

	return programs
}


func parseXMLTVTime(s string) (time.Time, error) {
	return time.Parse("20060102150405 -0700", s)
}

type xmlTV struct {
	XMLName    xml.Name    `xml:"tv"`
	Programmes []programme `xml:"programme"`
}

type programme struct {
	Channel string   `xml:"channel,attr"`
	Start   string   `xml:"start,attr"`
	Stop    string   `xml:"stop,attr"`
	Titles  []title  `xml:"title"`
}

type title struct {
	Lang  string `xml:"lang,attr"`
	Value string `xml:",chardata"`
}

func (p *provider) Timezone() string { return "Asia/Shanghai" }
