package whaletvplus

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
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

const (
	apiToken = "4ef13b5f3d2744e3b0a569feb8dde298"
	baseURL  = "https://rlaxx.zeasn.tv/livetv/api/device/browser/v1"
)

func init() {
	plugin.Register(&provider{
		client: httpclient.Browser(),
	})
}

type provider struct {
	client    httpClient
	authToken string
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "watch.whaletvplus.com" }
func (p *provider) Description() string { return "WhaleTV Plus free streaming service (362 channels, multi-language)" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "watch.whaletvplus.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if err := p.ensureAuth(); err != nil {
		return nil, err
	}

	start := date.Truncate(24 * time.Hour).UTC()
	end := start.Add(24 * time.Hour)

	url := fmt.Sprintf(
		"%s/epg?channelIds=%s&startTime=%d&endTime=%d",
		baseURL,
		channel.SiteID,
		start.UnixMilli(),
		end.UnixMilli(),
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	setHeaders(req, p.authToken)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from whaletvplus", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func (p *provider) ensureAuth() error {
	if p.authToken != "" {
		return nil
	}

	url := fmt.Sprintf("https://rlaxx.zeasn.tv/livetv/api/v1/auth/access?uuid=1&apiToken=%s&langCode=en", apiToken)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	setHeaders(req, "")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from whaletvplus", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result authResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if result.Data.Token == "" {
		return fmt.Errorf("whaletvplus: auth failed, no token returned")
	}

	p.authToken = result.Data.Token
	return nil
}

func setHeaders(req *http.Request, token string) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:147.0) Gecko/20100101 Firefox/147.0")
	req.Header.Set("Referer", "https://watch.whaletvplus.com/")
	req.Header.Set("Origin", "https://watch.whaletvplus.com")
	if token != "" {
		req.Header.Set("token", token)
	}
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		return nil, nil
	}

	ptList := response.Data[0].PtList
	programs := make([]domain.Program, 0, len(ptList))

	for _, item := range ptList {
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "watch.whaletvplus.com",
			Title:   item.PrgTitle,
			Lang:    channel.Lang,
		}

		if startMs, err := strconv.ParseInt(item.PrgStm, 10, 64); err == nil {
			p.Start = time.UnixMilli(startMs).UTC()
		}
		if endMs, err := strconv.ParseInt(item.PrgEtm, 10, 64); err == nil {
			p.Stop = time.UnixMilli(endMs).UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type authResponse struct {
	Data struct {
		Token string `json:"token"`
	} `json:"data"`
}

type apiResponse struct {
	Data []channelData `json:"data"`
}

type channelData struct {
	ChlID  string        `json:"chlId"`
	ChlNm  string        `json:"chlNm"`
	PtList []programItem `json:"ptList"`
}

type programItem struct {
	PrgEtm   string `json:"prgEtm"`
	PrgStm   string `json:"prgStm"`
	PrgTitle string `json:"prgTitle"`
	PrgchID  string `json:"prgchId"`
}

func (p *provider) Timezone() string { return "Asia/Seoul" }
