package magentatv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

const (
	authURL    = "https://api.prod.sngtv.magentatv.de/EPG/JSON/Authenticate"
	epgURL     = "https://api.prod.sngtv.magentatv.de/EPG/JSON/PlayBillList"
	sourceName = "web.magentatv.de"
)

var cookieNames = []string{"JSESSIONID", "CSESSIONID", "CSRFSESSION"}

func init() {
	plugin.Register(&provider{
		client: httpclient.Browser(),
	})
}

type provider struct {
	client httpClient
	mu     sync.Mutex
	token  string
	cookie string
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return sourceName }
func (p *provider) Description() string { return "Germany MagentaTV" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, sourceName)
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if err := p.authenticate(); err != nil {
		return nil, fmt.Errorf("auth failed: %w", err)
	}

	day := date.Truncate(24 * time.Hour).UTC()

	body := epgRequest{
		Count:         -1,
		IsFillProgram: 1,
		Offset:        0,
		Properties: []epgProperty{
			{
				Include: "endtime,genres,id,name,starttime,channelid,pictures,introduce,subName,seasonNum,subNum,cast,country,producedate,externalIds",
				Name:    "playbill",
			},
		},
		Type:      2,
		BeginTime: day.Format("20060102150405"),
		ChannelID: channel.SiteID,
		EndTime:   day.Add(24 * time.Hour).Format("20060102150405"),
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, epgURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("X_CSRFTOKEN", p.token)
	req.Header.Set("Cookie", p.cookie)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from magentatv", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel)
}

func (p *provider) authenticate() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.token != "" && p.cookie != "" {
		return nil
	}

	authBody := `{"terminalid":"00:00:00:00:00:00","mac":"00:00:00:00:00:00","terminaltype":"WEBTV","utcEnable":1,"timezone":"Etc/GMT0","userType":3,"terminalvendor":"Unknown"}`

	req, err := http.NewRequest(http.MethodPost, authURL+"?SID=firstup&T=Windows_chrome_118", strings.NewReader(authBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from magentatv", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var authResp authResponse
	if err := json.Unmarshal(data, &authResp); err != nil {
		return err
	}

	if authResp.CSRFToken == "" {
		return fmt.Errorf("no csrf token in auth response")
	}

	var cookies []string
	for _, header := range resp.Header["Set-Cookie"] {
		for _, name := range cookieNames {
			if idx := strings.Index(header, name+"="); idx >= 0 {
				end := strings.Index(header[idx:], ";")
				if end < 0 {
					cookies = append(cookies, header[idx:])
				} else {
					cookies = append(cookies, header[idx:idx+end+1])
				}
				break
			}
		}
	}

	p.token = authResp.CSRFToken
	p.cookie = strings.Join(cookies, " ")

	return nil
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response epgResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	if response.PlayBillList == nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(response.PlayBillList))
	for _, item := range response.PlayBillList {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      sourceName,
			Title:       item.Name,
			SubTitle:    item.SubName,
			Description: item.Introduce,
			Image:       parseImage(item.Pictures),
			Category:    parseCategory(item.Genres),
			Season:      parseInt(item.SeasonNum),
			Episode:     parseInt(item.SubNum),
			Lang:        channel.Lang,
		}

		if item.StartTime != "" {
			p.Start = parseTime(item.StartTime)
		}
		if item.EndTime != "" {
			p.Stop = parseTime(item.EndTime)
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func parseTime(s string) time.Time {
	if len(s) < 19 {
		return time.Time{}
	}
	t, _ := time.Parse("2006-01-02 15:04:05", s[:19])
	return t.UTC()
}

func parseImage(pictures []picture) string {
	if len(pictures) == 0 {
		return ""
	}
	return pictures[0].Href
}

func parseCategory(genres string) string {
	if genres == "" {
		return ""
	}
	replaced := strings.ReplaceAll(genres, "und", ",")
	parts := strings.Split(replaced, ",")
	trimmed := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			trimmed = append(trimmed, s)
		}
	}
	if len(trimmed) == 0 {
		return ""
	}
	return strings.Join(trimmed, ",")
}

func parseInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

type authResponse struct {
	CSRFToken string `json:"csrfToken"`
}

type epgRequest struct {
	Count         int           `json:"count"`
	IsFillProgram int           `json:"isFillProgram"`
	Offset        int           `json:"offset"`
	Properties    []epgProperty `json:"properties"`
	Type          int           `json:"type"`
	BeginTime     string        `json:"begintime"`
	ChannelID     string        `json:"channelid"`
	EndTime       string        `json:"endtime"`
}

type epgProperty struct {
	Include string `json:"include"`
	Name    string `json:"name"`
}

type epgResponse struct {
	PlayBillList []playBillItem `json:"playbilllist"`
}

type playBillItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Introduce   string    `json:"introduce"`
	ChannelID   string    `json:"channelid"`
	StartTime   string    `json:"starttime"`
	EndTime     string    `json:"endtime"`
	Genres      string    `json:"genres"`
	Pictures    []picture `json:"pictures"`
	SubName     string    `json:"subName"`
	SeasonNum   string    `json:"seasonNum"`
	SubNum      string    `json:"subNum"`
	Cast        *cast     `json:"cast"`
	Country     string    `json:"country"`
	ProduceDate string    `json:"producedate"`
	ExternalIDs string    `json:"externalIds"`
}

type picture struct {
	Href string `json:"href"`
}

type cast struct {
	Director string `json:"director"`
	Producer string `json:"producer"`
	Adaptor  string `json:"adaptor"`
	Actor    string `json:"actor"`
}
