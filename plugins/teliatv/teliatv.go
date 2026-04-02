package teliatv

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

func (p *provider) Name() string        { return "teliatv.ee" }
func (p *provider) Description() string { return "Telia TV Estonia" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "teliatv.ee")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	parts := strings.SplitN(channel.SiteID, "#", 2)
	lang := parts[0]
	channelID := parts[1]

	url := fmt.Sprintf(
		"https://api.teliatv.ee/dtv-api/3.2/%s/epg/guide?channelIds=%s&relations=programmes&images=webGuideItemLarge&startAt=%sT00:00&startAtOp=lte&endAt=%sT00:00&endAtOp=gt",
		lang, channelID,
		date.AddDate(0, 0, 1).Format("2006-01-02"),
		date.Format("2006-01-02"),
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
		return nil, fmt.Errorf("HTTP %d from teliatv", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	parts := strings.SplitN(channel.SiteID, "#", 2)
	channelID := ""
	if len(parts) == 2 {
		channelID = parts[1]
	}

	items, ok := response.CategoryItems[channelID]
	if !ok {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "teliatv.ee",
			Title:   item.Name,
			Lang:    channel.Lang,
		}

		if item.Images.WebGuideItemLarge != "" {
			p.Image = item.Images.WebGuideItemLarge
		}

		for _, progID := range item.Related.ProgrammeIDs {
			idStr := fmt.Sprintf("%d", progID)
			if prog, exists := response.Relations.Programmes[idStr]; exists {
				if t, err := time.Parse(time.RFC3339, prog.StartAt); err == nil {
					p.Start = t.UTC()
				}
				if t, err := time.Parse(time.RFC3339, prog.EndAt); err == nil {
					p.Stop = t.UTC()
				}
				break
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	CategoryItems map[string][]categoryItem `json:"categoryItems"`
	Relations     relations                 `json:"relations"`
}

type categoryItem struct {
	Name    string      `json:"name"`
	Images  imageData   `json:"images"`
	Related relatedData `json:"related"`
}

type imageData struct {
	WebGuideItemLarge string `json:"webGuideItemLarge"`
}

type relatedData struct {
	ProgrammeIDs []int64 `json:"programmeIds"`
}

type relations struct {
	Programmes map[string]programme `json:"programmes"`
}

type programme struct {
	StartAt string `json:"startAt"`
	EndAt   string `json:"endAt"`
}

func (p *provider) Timezone() string { return "Europe/Tallinn" }
