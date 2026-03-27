package tvmusorhu

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
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{
		client: http.DefaultClient,
	})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "tvmusor.hu" }
func (p *provider) Description() string { return "TV Musor Hungarian TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvmusor.hu")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	dateStr := date.Format("2006-01-02")
	formData := fmt.Sprintf(`data={"blocks":["%s|%s"]}`, channel.SiteID, dateStr)

	req, err := http.NewRequest(http.MethodPost, "https://tvmusor.borsonline.hu/a/get-events/", strings.NewReader(formData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

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
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	if response.Data.LoadedBlocks == nil {
		return []domain.Program{}, nil
	}

	blockID := fmt.Sprintf("%s_%s", channel.SiteID, date.Format("2006-01-02"))
	items, ok := response.Data.LoadedBlocks[blockID]
	if !ok {
		return []domain.Program{}, nil
	}

	seen := make(map[int64]bool)
	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		if seen[item.E] {
			continue
		}
		seen[item.E] = true

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tvmusor.hu",
			Title:       item.J,
			Description: item.C,
			Category:    item.H,
			Lang:        channel.Lang,
			Start:       time.UnixMilli(item.E).UTC(),
			Stop:        time.UnixMilli(item.F).UTC(),
		}

		if item.Z != "" {
			p.Image = fmt.Sprintf("https://tvmusor.borsonline.hu/images/events/408/%s", item.Z)
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Data responseData `json:"data"`
}

type responseData struct {
	LoadedBlocks map[string][]eventItem `json:"loadedBlocks"`
}

type eventItem struct {
	A int64  `json:"a"`
	B int64  `json:"b"`
	C string `json:"c"`
	D int    `json:"d"`
	E int64  `json:"e"`
	F int64  `json:"f"`
	H string `json:"h"`
	J string `json:"j"`
	Z string `json:"z"`
}
