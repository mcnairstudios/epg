package imjhnz

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
	"github.com/gavinmcnair/epg/internal/xmltv"
)

const apiEndpoint = "https://raw.githubusercontent.com/matthuisman/i.mjh.nz/master"

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "i.mjh.nz" }
func (p *provider) Description() string { return "Matt Huisman XMLTV aggregator (AU/NZ/Plex/Samsung/etc)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "imjhnz-plex-darkmatter", Name: "Dark Matter TV", Source: "i.mjh.nz", SiteID: "Plex/all#5e20b730f2f8d5003d739db7-5eea605674085f0040ddc7a6", Lang: "en", XmltvID: "DarkMatterTV.us"},
	}, nil
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel, channelID, date)
}

func Parse(data []byte, channel domain.Channel, channelID string, date time.Time) ([]domain.Program, error) {
	_, allPrograms, err := xmltv.Parse(data, "i.mjh.nz")
	if err != nil {
		return nil, err
	}

	day := date.Truncate(24 * time.Hour)
	nextDay := day.Add(24 * time.Hour)

	var filtered []domain.Program
	for _, p := range allPrograms {
		if p.Channel != channelID {
			continue
		}
		if p.Start.Before(day) || !p.Start.Before(nextDay) {
			continue
		}
		p.Channel = channel.XmltvID
		p.Source = "i.mjh.nz"
		filtered = append(filtered, p)
	}

	return mergeMovieParts(filtered), nil
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
