package taiwanplus

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

func (p *provider) Name() string        { return "taiwanplus.com" }
func (p *provider) Description() string { return "TaiwanPlus TV" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "taiwanplus.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	req, err := http.NewRequest(http.MethodGet, "https://www.taiwanplus.com/api/video/live/schedule/0", nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from taiwanplus", resp.StatusCode)
	}

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

	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return []domain.Program{}, nil
	}

	today := date.Truncate(24 * time.Hour).UTC()
	todayStr := today.Format("2006/01/02")

	programs := make([]domain.Program, 0)
	for _, sd := range resp.Data {
		if sd.Date != todayStr {
			continue
		}

		for i, item := range sd.Schedule {
			start, err := time.Parse("2006/01/02 15:04", item.DateTime)
			if err != nil {
				continue
			}

			var stop time.Time
			if i < len(sd.Schedule)-1 {
				stop, _ = time.Parse("2006/01/02 15:04", sd.Schedule[i+1].DateTime)
			} else {
				stop = start.Add(24 * time.Hour).Truncate(24 * time.Hour)
			}

			p := domain.Program{
				Channel:     channel.XmltvID,
				Source:      "taiwanplus.com",
				Title:       item.Title,
				Description: item.Description,
				Image:       item.Image,
				Category:    item.CategoryName,
				Lang:        channel.Lang,
				Start:       start.UTC(),
				Stop:        stop.UTC(),
			}

			programs = append(programs, p)
		}
	}

	return programs, nil
}

type apiResponse struct {
	Data []scheduleDate `json:"data"`
}

type scheduleDate struct {
	Date     string         `json:"date"`
	Schedule []scheduleItem `json:"schedule"`
}

type scheduleItem struct {
	DateTime     string `json:"dateTime"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Image        string `json:"image"`
	CategoryName string `json:"categoryName"`
	AgeRating    string `json:"ageRating"`
}

func (p *provider) Timezone() string { return "Asia/Taipei" }
