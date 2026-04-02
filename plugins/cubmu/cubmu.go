package cubmu

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

var jakartaTZ *time.Location

//go:embed channels.json
var channelsData []byte

func init() {
	jakartaTZ, _ = time.LoadLocation("Asia/Jakarta")
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

func (p *provider) Name() string        { return "cubmu.com" }
func (p *provider) Description() string { return "CubMu Indonesian TV streaming service" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "cubmu.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://servicebuss.transvision.co.id/v2/cms/getEPGData?app_id=cubmu&tvs_platform_id=standalone&schedule_date=%s&channel_id=%s",
		date.Format("2006-01-02"),
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
		return nil, fmt.Errorf("HTTP %d from cubmu", resp.StatusCode)
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

	trimmed := strings.TrimSpace(string(data))
	var response apiResponse
	if err := json.Unmarshal([]byte(trimmed), &response); err != nil {
		return []domain.Program{}, nil
	}

	if len(response.Result) == 0 {
		return []domain.Program{}, nil
	}

	lang := channel.Lang
	if lang == "" {
		lang = "id"
	}

	programs := make([]domain.Program, 0, len(response.Result))
	for _, item := range response.Result {
		start, _ := time.ParseInLocation("2006-01-02 15:04:05", item.ScheduleDate, jakartaTZ)
		datePart := strings.Split(item.ScheduleDate, " ")[0]
		stop, _ := time.ParseInLocation("2006-01-02 15:04:05", datePart+" "+item.ScheduleEndTime, jakartaTZ)

		description := item.ScheduleJSON.PrimarySynopsis
		if lang != "id" {
			description = item.ScheduleJSON.SecondarySynopsis
		}

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "cubmu.com",
			Title:       item.ScheduleTitle,
			Description: description,
			Lang:        lang,
			Start:       start.UTC(),
			Stop:        stop.UTC(),
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Result []scheduleItem `json:"result"`
}

type scheduleItem struct {
	ChannelID     string       `json:"channel_id"`
	ChannelName   string       `json:"channel_name"`
	ScheduleTitle string       `json:"scehedule_title"`
	ScheduleDate  string       `json:"schedule_date"`
	ScheduleEndTime string     `json:"schedule_end_time"`
	ScheduleJSON  scheduleJSON `json:"schedule_json"`
}

type scheduleJSON struct {
	PrimarySynopsis   string `json:"primarySynopsis"`
	SecondarySynopsis string `json:"secondarySynopsis"`
	EpisodeName       string `json:"episodeName"`
}

func (p *provider) Timezone() string { return "Asia/Jakarta" }
