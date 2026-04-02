package wavve

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

var seoulLocation *time.Location


func init() {
	seoulLocation, _ = time.LoadLocation("Asia/Seoul")
}

func (p *provider) Name() string        { return "wavve.com" }
func (p *provider) Description() string { return "Wavve Korean streaming EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "wavve.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	dateStr := date.Format("2006-01-02")
	nextDate := date.Add(24 * time.Hour).Format("2006-01-02")
	url := fmt.Sprintf(
		"https://apis.wavve.com/live/epgs/channels/%s?startdatetime=%s%%2000%%3A00&enddatetime=%s%%2000%%3A00&apikey=E5F3E0D30947AA5440556471321BB6D9&limit=500",
		channel.SiteID, dateStr, nextDate,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
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

	if len(resp.List) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(resp.List))
	for _, item := range resp.List {
		start, err := time.ParseInLocation("2006-01-02 15:04", item.StartTime, seoulLocation)
		if err != nil {
			continue
		}

		stop, err := time.ParseInLocation("2006-01-02 15:04", item.EndTime, seoulLocation)
		if err != nil {
			continue
		}

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "wavve.com",
			Title:   item.Title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		})
	}

	return programs, nil
}

type apiResponse struct {
	List []apiItem `json:"list"`
}

type apiItem struct {
	Title     string `json:"title"`
	StartTime string `json:"starttime"`
	EndTime   string `json:"endtime"`
}

func (p *provider) Timezone() string { return "Asia/Seoul" }
