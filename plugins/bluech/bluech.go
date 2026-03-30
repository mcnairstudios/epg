package bluech

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	plugin.Register(&provider{client: httpclient.Browser()})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "tv.blue.ch" }
func (p *provider) Description() string { return "Blue TV Switzerland" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tv.blue.ch")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Format("200601021504")
	end := date.Add(24 * time.Hour).Format("200601021504")

	url := fmt.Sprintf("https://services.sg101.prd.sctv.ch/catalog/tv/channels/list/(ids=%s;start=%s;end=%s;level=normal)",
		channel.SiteID, start, end)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from bluech", resp.StatusCode)
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
		return nil, nil
	}

	var broadcasts []broadcast
	for _, node := range resp.Nodes.Items {
		if node.Kind == "Channel" && node.Content.Nodes != nil {
			for _, item := range node.Content.Nodes.Items {
				if item.Kind == "Broadcast" {
					broadcasts = append(broadcasts, item)
				}
			}
		}
	}

	programs := make([]domain.Program, 0, len(broadcasts))
	for _, b := range broadcasts {
		title := b.Content.Description.Title
		if title == "Sendepause" {
			continue
		}

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tv.blue.ch",
			Title:       title,
			Description: b.Content.Description.Summary,
			Lang:        channel.Lang,
		}

		if len(b.Availabilities) > 0 {
			if t, err := time.Parse(time.RFC3339, b.Availabilities[0].AvailabilityStart); err == nil {
				p.Start = t.UTC()
			}
			if t, err := time.Parse(time.RFC3339, b.Availabilities[0].AvailabilityEnd); err == nil {
				p.Stop = t.UTC()
			}
		}

		if b.Content.Nodes != nil {
			for _, n := range b.Content.Nodes.Items {
				if n.Kind == "Image" && n.ContentPath != "" {
					p.Image = "https://services.sg101.prd.sctv.ch/content/images" + n.ContentPath + "_w1920.webp"
					break
				}
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Nodes struct {
		Items []channelNode `json:"Items"`
	} `json:"Nodes"`
}

type channelNode struct {
	Kind    string `json:"Kind"`
	Content struct {
		Nodes *struct {
			Items []broadcast `json:"Items"`
		} `json:"Nodes"`
	} `json:"Content"`
}

type broadcast struct {
	Kind    string `json:"Kind"`
	Content struct {
		Description struct {
			Title   string `json:"Title"`
			Summary string `json:"Summary"`
		} `json:"Description"`
		Nodes *struct {
			Items []struct {
				Kind        string `json:"Kind"`
				ContentPath string `json:"ContentPath"`
			} `json:"Items"`
		} `json:"Nodes"`
	} `json:"Content"`
	Availabilities []struct {
		AvailabilityStart string `json:"AvailabilityStart"`
		AvailabilityEnd   string `json:"AvailabilityEnd"`
	} `json:"Availabilities"`
}
