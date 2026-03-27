package anteltv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

var montevideo *time.Location

func init() {
	montevideo, _ = time.LoadLocation("America/Montevideo")
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

func (p *provider) Name() string        { return "anteltv.com.uy" }
func (p *provider) Description() string { return "Antel TV Uruguay guide" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "anteltv.com.uy")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	session, err := p.loadSession()
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	d := date.Format("2006-01-02")
	url := fmt.Sprintf(
		"https://cds-frontend.vera.com.uy/api-contenidos/canales/epg/%s?limit=500&dias_siguientes=0&fecha=%s&token=%s",
		channel.SiteID, d, session.Token,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+session.JWT)
	req.Header.Set("X-Frontend-Id", "1196")
	req.Header.Set("X-Service-Id", "3")
	req.Header.Set("X-System-Id", "1")

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

func (p *provider) loadSession() (*sessionResponse, error) {
	payload := []byte(`{"tipo":"anonima"}`)
	req, err := http.NewRequest(http.MethodPost, "https://veratv-be.vera.com.uy/api/sesiones", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var session sessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, err
	}
	return &session, nil
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response epgResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return []domain.Program{}, nil
	}

	if len(response.Data) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(response.Data))
	for _, item := range response.Data {
		start := parseTime(item.FechaHoraInicio)
		stop := parseTime(item.FechaHoraFin)
		if start.IsZero() || stop.IsZero() {
			continue
		}

		programs = append(programs, domain.Program{
			Channel:     channel.XmltvID,
			Source:      "anteltv.com.uy",
			Title:       item.NombrePrograma,
			SubTitle:    item.Subtitle,
			Description: item.DescripcionPrograma,
			Lang:        channel.Lang,
			Start:       start,
			Stop:        stop,
		})
	}

	return programs, nil
}

func parseTime(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, montevideo)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

type sessionResponse struct {
	Token string `json:"token"`
	JWT   string `json:"jwt"`
}

type epgResponse struct {
	Count int       `json:"count"`
	Data  []epgItem `json:"data"`
}

type epgItem struct {
	NombrePrograma      string `json:"nombre_programa"`
	DescripcionPrograma string `json:"descripcion_programa"`
	Subtitle            string `json:"subtitle"`
	FechaHoraInicio     string `json:"fecha_hora_inicio"`
	FechaHoraFin        string `json:"fecha_hora_fin"`
}
