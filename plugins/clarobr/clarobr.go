package clarobr

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

var saoPauloTZ *time.Location

func init() {
	var err error
	saoPauloTZ, err = time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		saoPauloTZ = time.FixedZone("America/Sao_Paulo", -3*60*60)
	}
}

//go:embed channels.json
var channelsData []byte

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

func (p *provider) Name() string        { return "claro.com.br" }
func (p *provider) Description() string { return "Claro TV Brazil" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "claro.com.br")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://programacao.claro.com.br/gatekeeper/exibicao/select?q=id_revel:(%s)+AND+id_cidade:1&wt=json&rows=100000&start=0&sort=id_canal+asc,dh_inicio+asc&fl=dh_fim+dh_inicio+st_titulo+titulo+id_programa+id_canal+id_cidade&fq=dh_inicio:[%s+TO+%s]",
		channel.SiteID,
		date.Format("2006-1-2T15:04:05Z"),
		date.AddDate(0, 0, 1).Add(-1*time.Minute).Format("2006-1-2T15:04:05Z"),
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response apiResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	programs := make([]domain.Program, 0, len(response.Response.Docs))
	for _, item := range response.Response.Docs {
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "claro.com.br",
			Title:   item.Titulo,
			Lang:    channel.Lang,
		}

		if t, err := parseTime(item.DhInicio); err == nil {
			p.Start = t.UTC()
		}
		if t, err := parseTime(item.DhFim); err == nil {
			p.Stop = t.UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

// parseTime parses time strings like "2025-8-13T00:00Z" in Sao Paulo timezone.
// The trailing Z is a literal character in this API, not a UTC indicator.
func parseTime(s string) (time.Time, error) {
	s = strings.TrimSuffix(s, "Z")
	return time.ParseInLocation("2006-1-2T15:04", s, saoPauloTZ)
}

type apiResponse struct {
	Response responseData `json:"response"`
}

type responseData struct {
	Docs []doc `json:"docs"`
}

type doc struct {
	Titulo  string `json:"titulo"`
	DhInicio string `json:"dh_inicio"`
	DhFim    string `json:"dh_fim"`
}
