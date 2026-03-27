package sibaco

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
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

func (p *provider) Name() string        { return "siba.com.co" }
func (p *provider) Description() string { return "SIBA Colombia TV listings" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "siba.com.co")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour).UTC()
	end := start.Add(24 * time.Hour)

	form := url.Values{}
	form.Set("servicio", "10")
	form.Set("ini", fmt.Sprint(start.Unix()))
	form.Set("end", fmt.Sprint(end.Unix()))
	form.Set("chn", channel.SiteID)

	encoded := form.Encode()
	req, err := http.NewRequest(http.MethodPost, "http://devportal.siba.com.co/index.php?action=grilla", strings.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

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
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, nil
	}

	if response.List == nil {
		return nil, nil
	}

	var channelData *channelList
	for _, item := range response.List {
		if item.ID == channel.SiteID {
			channelData = &item
			break
		}
	}

	if channelData == nil {
		return nil, nil
	}

	programs := make([]domain.Program, 0, len(channelData.Prog))
	for _, item := range channelData.Prog {
		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "siba.com.co",
			Title:   item.Nom,
			Lang:    channel.Lang,
			Start:   time.Unix(item.Ini, 0).UTC(),
			Stop:    time.Unix(item.Fin, 0).UTC(),
		})
	}

	return programs, nil
}

type apiResponse struct {
	List []channelList `json:"list"`
}

type channelList struct {
	ID   string    `json:"id"`
	Prog []program `json:"prog"`
}

type program struct {
	Nom string `json:"nom"`
	Ini int64  `json:"ini"`
	Fin int64  `json:"fin"`
}
