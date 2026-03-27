package myjcom

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

var tokyoTZ *time.Location

//go:embed channels.json
var channelsData []byte

var genres = map[int]string{
	0:  "ニュース／報道",
	1:  "スポーツ",
	2:  "情報／ワイドショー",
	3:  "ドラマ",
	4:  "音楽",
	5:  "バラエティ",
	6:  "映画",
	7:  "アニメ／特撮",
	8:  "ドキュメンタリー／教養",
	9:  "劇場／公演",
	10: "趣味／教育",
	11: "福祉",
	12: "その他",
}

func init() {
	tokyoTZ, _ = time.LoadLocation("Asia/Tokyo")
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

func (p *provider) Name() string        { return "tvguide.myjcom.jp" }
func (p *provider) Description() string { return "J:COM TV guide for Japanese cable TV" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "tvguide.myjcom.jp")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	id := fmt.Sprintf("%s_%s", channel.SiteID, date.Format("20060102"))
	url := fmt.Sprintf("https://tvguide.myjcom.jp/api/getEpgInfo/?channels=%s&rectime=&rec4k=", id)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", "AD_NAV=1; area_id=108;")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0 WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.103 Safari/537.36")

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
	id := fmt.Sprintf("%s_%s", channel.SiteID, date.Format("20060102"))

	var response map[string][]epgItem
	if err := json.Unmarshal(data, &response); err != nil {
		return []domain.Program{}, nil
	}

	items, ok := response[id]
	if !ok {
		return []domain.Program{}, nil
	}

	var programs []domain.Program
	for _, item := range items {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "tvguide.myjcom.jp",
			Title:       item.Title,
			Description: item.Commentary,
			Lang:        channel.Lang,
		}

		if item.ImgPath != "" {
			p.Image = "https://tvguide.myjcom.jp" + item.ImgPath
		}

		if item.SortGenre != "" {
			genreChar := string(item.SortGenre[0])
			if genreID, err := strconv.Atoi(genreChar); err == nil {
				if name, ok := genres[genreID]; ok {
					p.Category = name
				}
			}
		}

		startStr := strconv.FormatInt(item.ProgramStart, 10)
		if t, err := time.ParseInLocation("20060102150405", startStr, tokyoTZ); err == nil {
			p.Start = t.UTC()
		}

		stopStr := strconv.FormatInt(item.ProgramEnd, 10)
		if t, err := time.ParseInLocation("20060102150405", stopStr, tokyoTZ); err == nil {
			p.Stop = t.UTC()
		}

		programs = append(programs, p)
	}

	if programs == nil {
		programs = []domain.Program{}
	}

	return programs, nil
}

type epgItem struct {
	Title        string   `json:"title"`
	Commentary   string   `json:"commentary"`
	ImgPath      string   `json:"imgPath"`
	SortGenre    string   `json:"sortGenre"`
	ProgramStart int64    `json:"programStart"`
	ProgramEnd   int64    `json:"programEnd"`
}
