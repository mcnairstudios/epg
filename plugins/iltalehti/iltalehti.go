package iltalehti

import (
	"encoding/json"
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

func (p *provider) Name() string        { return "iltalehti.fi" }
func (p *provider) Description() string { return "Iltalehti Finnish TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "iltalehti.fi")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	parts := strings.SplitN(channel.SiteID, "#", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid site_id format: %s", channel.SiteID)
	}
	group := parts[0]

	d := date.Truncate(24 * time.Hour).UTC()
	end := d.Add(24 * time.Hour)

	url := fmt.Sprintf(
		"https://il-telkku-api.prod.il.fi/v1/channel-groups/%s/offering?startTime=%s&endTime=%s",
		group, d.Format(time.RFC3339), end.Format(time.RFC3339),
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
		return nil, fmt.Errorf("HTTP %d from iltalehti", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return []domain.Program{}, nil
	}

	if resp.Response == nil {
		return []domain.Program{}, nil
	}

	parts := strings.SplitN(channel.SiteID, "#", 2)
	if len(parts) != 2 {
		return []domain.Program{}, nil
	}
	channelID := parts[1]

	var channelData *channelOffering
	for i := range resp.Response {
		if resp.Response[i].ChannelID == channelID {
			channelData = &resp.Response[i]
			break
		}
	}

	if channelData == nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0)
	for _, timeSlot := range channelData.Programs {
		for _, item := range timeSlot {
			p := domain.Program{
				Channel:     channel.XmltvID,
				Source:      "iltalehti.fi",
				Title:       item.ProgramName,
				Description: item.Description,
				Season:      item.SeasonNumber,
				Episode:     item.EpisodeNumber,
				Lang:        channel.Lang,
			}

			for _, img := range item.Images {
				if img.SizeTag == "612x382" {
					p.Image = img.URL
					break
				}
			}

			if item.StartTime != "" {
				if t, err := time.Parse(time.RFC3339Nano, item.StartTime); err == nil {
					p.Start = t.UTC()
				}
			}
			if item.EndTime != "" {
				if t, err := time.Parse(time.RFC3339Nano, item.EndTime); err == nil {
					p.Stop = t.UTC()
				}
			}

			programs = append(programs, p)
		}
	}

	return programs, nil
}

type apiResponse struct {
	Response []channelOffering `json:"response"`
}

type channelOffering struct {
	ChannelID string                     `json:"channelId"`
	Programs  map[string][]programEntry  `json:"programs"`
}

type programEntry struct {
	ProgramName   string      `json:"programName"`
	Description   string      `json:"description"`
	SeasonNumber  int         `json:"seasonNumber"`
	EpisodeNumber int         `json:"episodeNumber"`
	StartTime     string      `json:"startTime"`
	EndTime       string      `json:"endTime"`
	Images        []imageInfo `json:"images"`
}

type imageInfo struct {
	SizeTag string `json:"sizeTag"`
	URL     string `json:"url"`
}

func (p *provider) Timezone() string { return "Europe/Helsinki" }
