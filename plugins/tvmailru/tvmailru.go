package tvmailru

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "tv.mail.ru" }
func (p *provider) Description() string { return "Mail.ru TV Russia EPG" }

var moscowLocation *time.Location


func init() {
	moscowLocation, _ = time.LoadLocation("Europe/Moscow")
}

var categories = map[int]string{
	1:  "Фильм",
	2:  "Сериал",
	6:  "Документальное",
	7:  "Телемагазин",
	8:  "Позновательное",
	10: "Другое",
	14: "ТВ-шоу",
	16: "Досуг,Хобби",
	17: "Ток-шоу",
	18: "Юмористическое",
	23: "Музыка",
	24: "Развлекательное",
	25: "Игровое",
	26: "Новости",
}

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tv.mail.ru")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://tv.mail.ru/ajax/channel/?region_id=70&channel_id=%s&date=%s",
		channel.SiteID,
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, date)
}

func Parse(data []byte, date time.Time) ([]domain.Program, error) {
	items := parseItems(data)
	if len(items) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(items))
	currentDate := date

	for i, item := range items {
		dateStr := fmt.Sprintf("%s %s", currentDate.Format("2006-01-02"), item.Start)
		start, err := time.ParseInLocation("2006-01-02 15:04", dateStr, moscowLocation)
		if err != nil {
			continue
		}
		start = start.UTC()

		if len(programs) > 0 && start.Before(programs[len(programs)-1].Start) {
			start = start.Add(24 * time.Hour)
			currentDate = currentDate.Add(24 * time.Hour)
		}

		stop := start.Add(1 * time.Hour)

		if len(programs) > 0 {
			programs[len(programs)-1].Stop = start
		}

		var category string
		if cat, ok := categories[item.CategoryID]; ok {
			category = cat
		}

		_ = i

		programs = append(programs, domain.Program{
			Source:   "tv.mail.ru",
			Title:    item.Name,
			Category: category,
			Lang:     "ru",
			Start:    start,
			Stop:     stop,
		})
	}

	return programs, nil
}

func parseItems(data []byte) []event {
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil
	}
	if len(resp.Schedule) == 0 || resp.Schedule[0].Event == nil {
		return nil
	}
	e := resp.Schedule[0].Event
	var result []event
	result = append(result, e.Past...)
	result = append(result, e.Current...)
	return result
}

type apiResponse struct {
	Schedule []scheduleItem `json:"schedule"`
}

type scheduleItem struct {
	Event *eventData `json:"event"`
}

type eventData struct {
	Past    []event `json:"past"`
	Current []event `json:"current"`
}

type event struct {
	Name       string `json:"name"`
	Start      string `json:"start"`
	CategoryID int    `json:"category_id"`
}
