package zapao

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

func (p *provider) Name() string        { return "zap.co.ao" }
func (p *provider) Description() string { return "ZAP Angola TV listings" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "zap.co.ao")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://zapon.zapsi.net/ao/m/api/epg/events?date=%s&channel=%s",
		date.Format("20060102"),
		channel.SiteID,
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
		return []domain.Program{}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return []domain.Program{}, nil
	}

	if len(response.Data) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(response.Data))
	for _, item := range response.Data {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "zap.co.ao",
			Title:       item.ProgramName,
			Description: item.ProgramDescription,
			Category:    item.CategoryName,
			Lang:        channel.Lang,
			Start:       time.Unix(int64(item.UTCBeginDate), 0).UTC(),
			Stop:        time.Unix(int64(item.UTCEndDate), 0).UTC(),
		}
		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Data []eventItem `json:"data"`
}

type eventItem struct {
	ProgramName        string `json:"programName"`
	ProgramDescription string `json:"programDescription"`
	CategoryName       string `json:"categoryName"`
	UTCBeginDate       int    `json:"utcBeginDate"`
	UTCEndDate         int    `json:"utcEndDate"`
}

func (p *provider) Timezone() string { return "Europe/Lisbon" }
