package t2net

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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
	apiLocale  = "sl-SI"
	apiVersion = "10.8"
	apiFormat  = "json"
	apiUUID    = "7971845803564301055"
	apiToken   = "f8cce4e4-3ccc-486d-ac02-73cf231e3a2b"
)

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

func (p *provider) Name() string        { return "tv2go.t-2.net" }
func (p *provider) Description() string { return "T-2 Slovenia (342 channels)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tv2go.t-2.net")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	channelID, _ := strconv.Atoi(channel.SiteID)

	body := map[string]interface{}{
		"locale":           apiLocale,
		"channelId":        []int{channelID},
		"startTime":        date.UnixMilli(),
		"endTime":          date.Add(24 * time.Hour).UnixMilli(),
		"imageInfo":        []map[string]int{{"height": 500, "width": 1100}},
		"includeBookmarks": false,
		"includeShow":      true,
	}

	endpoint := "client/tv/getEpg"
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	hash := generateHash(string(bodyBytes), endpoint)
	url := fmt.Sprintf("https://tv2go.t-2.net/Catherine/api/%s/%s/%s/%s/%s",
		apiVersion, apiFormat, apiUUID, hash, endpoint)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from t2net", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return []domain.Program{}, nil
	}

	if response.Entries == nil {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(response.Entries))
	for _, item := range response.Entries {
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "tv2go.t-2.net",
			Title:   item.Name,
			Lang:    channel.Lang,
		}

		if item.Show != nil {
			p.Description = item.Show.ShortDescription
			if len(item.Show.Genres) > 0 {
				p.Category = item.Show.Genres[0].Name
			}
		}

		if len(item.Images) > 0 && item.Images[0].URL != "" {
			p.Image = "https://tv2go.t-2.net" + item.Images[0].URL
		}

		if ts, err := strconv.ParseInt(item.StartTimestamp, 10, 64); err == nil {
			p.Start = time.UnixMilli(ts).UTC()
		}
		if ts, err := strconv.ParseInt(item.EndTimestamp, 10, 64); err == nil {
			p.Stop = time.UnixMilli(ts).UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

func generateHash(body string, endpoint string) string {
	salt := apiToken + apiVersion + apiFormat + apiUUID
	h := md5.Sum([]byte(salt + endpoint + body))
	return hex.EncodeToString(h[:])
}

type apiResponse struct {
	Entries []epgEntry `json:"entries"`
}

type epgEntry struct {
	ChannelID      int           `json:"channelId"`
	StartTimestamp string        `json:"startTimestamp"`
	EndTimestamp   string        `json:"endTimestamp"`
	Name           string        `json:"name"`
	Images         []entryImage  `json:"images"`
	Show           *showDetail   `json:"show"`
}

type entryImage struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type showDetail struct {
	ShortDescription string       `json:"shortDescription"`
	Genres           []genreEntry `json:"genres"`
}

type genreEntry struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
