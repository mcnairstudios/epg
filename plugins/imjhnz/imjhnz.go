package imjhnz

import (
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
	"github.com/gavinmcnair/epg/internal/xmltv"
)

const apiEndpoint = "https://raw.githubusercontent.com/matthuisman/i.mjh.nz/master"

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{client: httpclient.Browser(), docs: newDocs()})
}

type provider struct {
	client httpClient
	docs   *fetchcache.Cache[doc]
}

type doc map[string][]domain.Program

func newDocs() *fetchcache.Cache[doc] {
	return fetchcache.New(fetchcache.Options[doc]{
		TTL:        fetchcache.DocTTL[doc](15 * time.Minute),
		MaxEntries: 4,
	})
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "i.mjh.nz" }
func (p *provider) Description() string { return "Matt Huisman XMLTV aggregator (AU/NZ/Plex/Samsung/etc)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	chs, err := channels.Load(channelsData, "i.mjh.nz")
	if err != nil {
		return nil, err
	}
	return channels.GroupBy(chs, func(c domain.Channel) string {
		path, _, _ := splitSiteID(c.SiteID)
		return path
	}), nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	path, channelID, ok := splitSiteID(channel.SiteID)
	if !ok {
		return nil, fmt.Errorf("invalid site_id format: %s", channel.SiteID)
	}

	d, err := p.docs.Get(path, func() (doc, error) { return p.download(path) })
	if err != nil {
		return nil, err
	}
	return selectPrograms(d, channel, channelID, date), nil
}

func (p *provider) download(path string) (doc, error) {
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
		return nil, &fetchcache.StatusError{Code: resp.StatusCode, Source: "imjhnz"}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return buildDoc(body)
}

func Parse(data []byte, channel domain.Channel, channelID string, date time.Time) ([]domain.Program, error) {
	d, err := buildDoc(data)
	if err != nil {
		return nil, err
	}
	return selectPrograms(d, channel, channelID, date), nil
}

func buildDoc(data []byte) (doc, error) {
	_, allPrograms, err := xmltv.Parse(data, "i.mjh.nz")
	if err != nil {
		return nil, err
	}
	d := make(doc)
	for _, p := range allPrograms {
		d[p.Channel] = append(d[p.Channel], p)
	}
	return d, nil
}

func selectPrograms(d doc, channel domain.Channel, channelID string, date time.Time) []domain.Program {
	day := date.Truncate(24 * time.Hour)
	nextDay := day.Add(24 * time.Hour)

	var filtered []domain.Program
	for _, p := range d[channelID] {
		if p.Start.Before(day) || !p.Start.Before(nextDay) {
			continue
		}
		p.Channel = channel.XmltvID
		p.Source = "i.mjh.nz"
		filtered = append(filtered, p)
	}

	return mergeMovieParts(filtered)
}

func mergeMovieParts(programs []domain.Program) []domain.Program {
	if len(programs) == 0 {
		return programs
	}

	var output []domain.Program
	for _, p := range programs {
		if len(output) > 0 && p.Category == "Movie" {
			prev := &output[len(output)-1]
			if prev.Title == p.Title && prev.Description == p.Description {
				prev.Stop = p.Stop
				continue
			}
		}
		output = append(output, p)
	}
	return output
}

func splitSiteID(siteID string) (string, string, bool) {
	parts := strings.SplitN(siteID, "#", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func (p *provider) Timezone() string { return "Australia/Sydney" }
