package epg112114

import (
	"encoding/xml"
	"io"
	"net/http"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "epg.112114.xyz" }
func (p *provider) Description() string { return "Chinese IPTV EPG from 112114.xyz" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "epg112114-btv-wenyi", Name: "BTV文艺", Source: "epg.112114.xyz", SiteID: "BTV文艺", Lang: "zh", XmltvID: "BRTVArtsChannel.cn"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
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

	return programs, nil
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
