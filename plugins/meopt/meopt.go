package meopt

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

var lisbonTZ *time.Location

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

func (p *provider) Name() string        { return "meo.pt" }
func (p *provider) Description() string { return "MEO Portuguese TV provider" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "meopt-RTPM", Name: "RTP Madeira", Source: "meo.pt", SiteID: "RTPM", Lang: "pt", XmltvID: "RTPMadeira.pt"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	u := fmt.Sprintf(
		"https://meogouser.apps.meo.pt/Services/GridTv/GridTv.svc/GetLiveChannelProgramsByDate?callLetter=%s&date=%s&userAgent=IPTV_OFR_GTV",
		channel.SiteID,
		date.Format("2006-01-02"),
	)

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Origin", "https://www.meo.pt")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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

	if len(response.Result) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(response.Result))
	for _, item := range response.Result {
		title := item.Title
		if title == "" {
			title = "Sem título"
		}

		start, _ := time.ParseInLocation("2006-01-02T15:04:05", item.StartDate, lisbonTZ)
		stop, _ := time.ParseInLocation("2006-01-02T15:04:05", item.EndDate, lisbonTZ)

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
