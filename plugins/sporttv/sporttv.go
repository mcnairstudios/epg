package sporttv

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
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

func (p *provider) Name() string        { return "sporttv.pt" }
func (p *provider) Description() string { return "Sport TV Portugal" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "sporttv-727", Name: "Sport TV 1", Source: "sporttv.pt", SiteID: "727", Lang: "pt", XmltvID: "SportTV1.pt"},
	}, nil
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	req, err := http.NewRequest(http.MethodGet, "https://www.sporttv.pt/guia", nil)
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

	return Parse(body, channel, date)
}

var nuxtRe = regexp.MustCompile(`<script id="__NUXT_DATA__"[^>]*>([\s\S]*?)</script>`)

func Parse(data []byte, channel domain.Channel, date time.Time) ([]domain.Program, error) {
	if len(data) == 0 {
		return []domain.Program{}, nil
	}

	m := nuxtRe.FindSubmatch(data)
	if m == nil {
		return []domain.Program{}, nil
	}

	var raw []json.RawMessage
	if err := json.Unmarshal(m[1], &raw); err != nil {
		return []domain.Program{}, nil
	}

	vals := make([]interface{}, len(raw))
	for i, r := range raw {
		var v interface{}
		json.Unmarshal(r, &v)
		vals[i] = v
	}

	if len(vals) < 3 {
		return []domain.Program{}, nil
	}

	dataIdx := getInt(vals, 1, "data")
	if dataIdx < 0 || dataIdx >= len(vals) {
		return []domain.Program{}, nil
	}

	dataMap, ok := vals[dataIdx].(map[string]interface{})
	if !ok {
		return []domain.Program{}, nil
	}

	keys := make([]string, 0, len(dataMap))
	for k := range dataMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) < 4 {
		return []domain.Program{}, nil
	}

	epgIdxRaw, ok := dataMap[keys[3]]
	if !ok {
		return []domain.Program{}, nil
	}
	epgIdx := int(epgIdxRaw.(float64))
	if epgIdx < 0 || epgIdx >= len(vals) {
		return []domain.Program{}, nil
	}

	epgArr, ok := vals[epgIdx].([]interface{})
	if !ok {
		return []domain.Program{}, nil
	}

	chID, _ := strconv.Atoi(channel.SiteID)
	d := date.Truncate(24 * time.Hour).UTC()

	programs := make([]domain.Program, 0)
	for _, rawIdx := range epgArr {
		idx := int(rawIdx.(float64))
		if idx < 0 || idx >= len(vals) {
			continue
		}
		obj, ok := vals[idx].(map[string]interface{})
		if !ok {
			continue
		}

		e := resolve(obj, vals)
		if e.canalID != chID {
			continue
		}

		st := time.UnixMilli(e.data).UTC()
		if st.Year() != d.Year() || st.Month() != d.Month() || st.Day() != d.Day() {
			continue
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "sporttv.pt",
			Title:       e.descricao,
			Description: e.evento,
			Image:       e.imagem,
			Category:    e.modalidade,
			Lang:        channel.Lang,
			Start:       st,
			Stop:        st.Add(time.Duration(e.duracao) * time.Millisecond),
		})
	}

	return programs, nil
}

type entry struct {
	data       int64
	duracao    int64
	descricao  string
	imagem     string
	canalID    int
	modalidade string
	evento     string
}

func resolve(obj map[string]interface{}, vals []interface{}) entry {
	var e entry
	rv := func(key string) interface{} {
		raw, ok := obj[key]
		if !ok {
			return nil
		}
		idx := int(raw.(float64))
		if idx >= 0 && idx < len(vals) {
			return vals[idx]
		}
		return nil
	}
	if v := rv("data"); v != nil {
		if f, ok := v.(float64); ok {
			e.data = int64(f)
		}
	}
	if v := rv("duracao"); v != nil {
		if f, ok := v.(float64); ok {
			e.duracao = int64(f)
		}
	}
	if v := rv("descricao"); v != nil {
		if s, ok := v.(string); ok {
			e.descricao = s
		}
	}
	if v := rv("imagem"); v != nil {
		if s, ok := v.(string); ok {
			e.imagem = s
		}
	}
	if raw, ok := obj["canal"]; ok {
		ci := int(raw.(float64))
		if ci >= 0 && ci < len(vals) {
			if cm, ok := vals[ci].(map[string]interface{}); ok {
				if idRaw, ok := cm["id"]; ok {
					ii := int(idRaw.(float64))
					if ii >= 0 && ii < len(vals) {
						if f, ok := vals[ii].(float64); ok {
							e.canalID = int(f)
						}
					}
				}
			}
		}
	}
	if raw, ok := obj["modalidade"]; ok {
		mi := int(raw.(float64))
		if mi >= 0 && mi < len(vals) {
			if mm, ok := vals[mi].(map[string]interface{}); ok {
				if nr, ok := mm["nomeModalidade"]; ok {
					ni := int(nr.(float64))
					if ni >= 0 && ni < len(vals) {
						if s, ok := vals[ni].(string); ok {
							e.modalidade = s
						}
					}
				}
			}
		}
	}
	if raw, ok := obj["evento"]; ok {
		ei := int(raw.(float64))
		if ei >= 0 && ei < len(vals) {
			if em, ok := vals[ei].(map[string]interface{}); ok {
				if nr, ok := em["nome"]; ok {
					ni := int(nr.(float64))
					if ni >= 0 && ni < len(vals) {
						if s, ok := vals[ni].(string); ok {
							e.evento = s
						}
					}
				}
			}
		}
	}
	return e
}

func getInt(vals []interface{}, idx int, key string) int {
	if idx < 0 || idx >= len(vals) {
		return -1
	}
	m, ok := vals[idx].(map[string]interface{})
	if !ok {
		return -1
	}
	v, ok := m[key]
	if !ok {
		return -1
	}
	return int(v.(float64))
}
