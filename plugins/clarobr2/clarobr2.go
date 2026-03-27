package clarobr2

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
	plugin.Register(&provider{client: http.DefaultClient})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "claro.com.br" }
func (p *provider) Description() string { return "Claro Brazil EPG" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "claro.com.br")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	start := date.Truncate(24 * time.Hour)
	end := start.Add(24*time.Hour - time.Minute)

	url := fmt.Sprintf(
		"https://programacao.claro.com.br/gatekeeper/exibicao/select?q=id_revel:(%s)+AND+id_cidade:1&wt=json&rows=100000&start=0&sort=id_canal+asc,dh_inicio+asc&fl=dh_fim+dh_inicio+st_titulo+titulo+id_programa+id_canal+id_cidade&fq=dh_inicio:[%s+TO+%s]",
		channel.SiteID,
		formatClaroTime(start),
		formatClaroTime(end),
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

	return Parse(body, channel)
}

func formatClaroTime(t time.Time) string {
	return fmt.Sprintf("%d-%d-%dT%02d:%02d:%02dZ",
		t.Year(), int(t.Month()), t.Day(),
		t.Hour(), t.Minute(), t.Second())
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return []domain.Program{}, nil
	}

	if len(resp.Response.Docs) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(resp.Response.Docs))
	for _, item := range resp.Response.Docs {
		start := parseTime(item.Start)
		stop := parseTime(item.Stop)
		if start.IsZero() || stop.IsZero() {
			continue
		}

		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "claro.com.br",
			Title:   item.Title,
			Lang:    channel.Lang,
			Start:   start,
			Stop:    stop,
		})
	}

	return programs, nil
}

func parseTime(s string) time.Time {
	s = strings.TrimSuffix(s, "Z")
	layouts := []string{
		"2006-1-2T15:04:05",
		"2006-1-2T15:04",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, saoPauloTZ); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

type apiResponse struct {
	Response responseBody `json:"response"`
}

type responseBody struct {
	Docs []docItem `json:"docs"`
}

type docItem struct {
	Title string `json:"titulo"`
	Start string `json:"dh_inicio"`
	Stop  string `json:"dh_fim"`
}
