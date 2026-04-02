package mxtvjp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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

var tokyo *time.Location

func init() {
	tokyo, _ = time.LoadLocation("Asia/Tokyo")
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

func (p *provider) Name() string        { return "s.mxtv.jp" }
func (p *provider) Description() string { return "Tokyo MX TV guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "s.mxtv.jp")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	d := date.Format("20060102")
	id := fmt.Sprintf("SV%sEPG%s", channel.SiteID, d)
	url := fmt.Sprintf("https://s.mxtv.jp/bangumi_file/json01/%s.json", id)

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
		return nil, fmt.Errorf("HTTP %d from mxtvjp", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var items []epgItem
	if err := json.Unmarshal(data, &items); err != nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		start := parseJapaneseTime(item.StartTime)
		if start.IsZero() {
			continue
		}
		stop := addDuration(start, item.Duration)

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "s.mxtv.jp",
			Title:       item.EventName,
			Description: item.EventText,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

func parseJapaneseTime(s string) time.Time {
	s = strings.ReplaceAll(s, "年", "-")
	s = strings.ReplaceAll(s, "月", "-")
	s = strings.ReplaceAll(s, "日", " ")
	s = strings.ReplaceAll(s, "時", ":")
	s = strings.ReplaceAll(s, "分", ":")
	s = strings.ReplaceAll(s, "秒", "")

	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, tokyo)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func addDuration(start time.Time, dur string) time.Time {
	parts := strings.Split(dur, ":")
	if len(parts) != 3 {
		return start
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	s, _ := strconv.Atoi(parts[2])
	return start.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s)*time.Second)
}

type epgItem struct {
	EventName string `json:"Event_name"`
	EventText string `json:"Event_text"`
	StartTime string `json:"Start_time"`
	Duration  string `json:"Duration"`
}

func (p *provider) Timezone() string { return "Asia/Tokyo" }
