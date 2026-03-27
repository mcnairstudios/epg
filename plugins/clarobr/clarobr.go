package clarobr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

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

func (p *provider) Name() string        { return "claro.com.br" }
func (p *provider) Description() string { return "Claro TV Brazil" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "claro-1682", Name: "Claro BR", Source: "claro.com.br", SiteID: "1_1682", Lang: "pt", XmltvID: "ClaroBR.br"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://programacao.claro.com.br/gatekeeper/exibicao/select?q=id_revel:(%s)+AND+id_cidade:1&wt=json&rows=100000&start=0&sort=id_canal+asc,dh_inicio+asc&fl=dh_fim+dh_inicio+st_titulo+titulo+id_programa+id_canal+id_cidade&fq=dh_inicio:[%s+TO+%s]",
		channel.SiteID,
		date.Format("2006-1-2T15:04:05Z"),
		date.AddDate(0, 0, 1).Format("2006-1-2T15:04:05Z"),
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

		if t, err := time.Parse(time.RFC3339, item.DhInicio); err == nil {
			p.Start = t.UTC()
		}
		if t, err := time.Parse(time.RFC3339, item.DhFim); err == nil {
			p.Stop = t.UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
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
