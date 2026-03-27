package meopt

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

var lisbonTZ *time.Location

//go:embed channels.json
var channelsData []byte

func init() {
	lisbonTZ, _ = time.LoadLocation("Europe/Lisbon")
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

func NewProvider(client httpClient) *provider {
	return &provider{client: client}
}

func (p *provider) Name() string        { return "meo.pt" }
func (p *provider) Description() string { return "MEO Portuguese TV provider" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "meo.pt")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	u := fmt.Sprintf(
		"https://meogouser.apps.meo.pt/Services/GridTv/GridTv.svc/GetLiveChannelProgramsByDate?callLetter=%s&date=%s&userAgent=IPTV_OFR_GTV",
		url.QueryEscape(channel.SiteID),
		date.Format("2006-01-02"),
	)

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func setHeaders(req *http.Request) {
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en-US;q=0.8,en;q=0.7,pt;q=0.6,cs;q=0.5")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Origin", "https://www.meo.pt")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Referer", "https://www.meo.pt/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Mobile Safari/537.36")
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil
	}

	if response.StatusCode != 200 || len(response.Result) == 0 {
		return nil, nil
	}

	programs := make([]domain.Program, 0, len(response.Result))
	for _, item := range response.Result {
		start, err := time.ParseInLocation("2006-01-02T15:04:05", item.StartDate, lisbonTZ)
		if err != nil {
			continue
		}
		stop, err := time.ParseInLocation("2006-01-02T15:04:05", item.EndDate, lisbonTZ)
		if err != nil {
			continue
		}

		title := item.Title
		if title == "" {
			title = "Sem título"
		}

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "meo.pt",
			Title:   title,
			Lang:    channel.Lang,
			Start:   start.UTC(),
			Stop:    stop.UTC(),
		}

		if item.Synopsis != "" {
			p.Description = item.Synopsis
		}

		if item.Title != "" && channel.SiteID != "" {
			encodedTitle := url.QueryEscape(item.Title)
			p.Image = fmt.Sprintf(
				"https://cdn-er-images.online.meo.pt/eemstb/ImageHandler.ashx?chCallLetter=%s&progTitle=%s&profile=16_9&profileFallback=false&noFallback=true&appSource=PC_CHROME_PWA&width=1920&csf",
				channel.SiteID,
				encodedTitle,
			)
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Status     string       `json:"Status"`
	StatusCode int          `json:"StatusCode"`
	Result     []programItem `json:"Result"`
}

type programItem struct {
	CallLetter string `json:"CallLetter"`
	StartDate  string `json:"StartDate"`
	EndDate    string `json:"EndDate"`
	Title      string `json:"Title"`
	Synopsis   string `json:"Synopsis"`
}
