package vidio

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/httpclient"
	"github.com/gavinmcnair/epg/internal/plugin"
)

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

func (p *provider) Name() string        { return "vidio.com" }
func (p *provider) Description() string { return "Vidio.com Indonesian streaming service" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "vidio.com")
}

var (
	webClientSecret = []byte("dPr0QImQ7bc5o9LMntNba2DOsSbZcjUh")
	webClientIV     = []byte("C8RWsrtFsoeyCyPt")
)

func (p *provider) getAPIKey() (string, error) {
	req, err := http.NewRequest(http.MethodPost, "https://www.vidio.com/auth", bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d from vidio", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var session struct {
		APIKey string `json:"api_key"`
	}
	if err := json.Unmarshal(body, &session); err != nil {
		return "", err
	}

	if session.APIKey == "" {
		return "", fmt.Errorf("no api_key in auth response")
	}

	// Encrypt the API key using AES-256-CBC
	block, err := aes.NewCipher(webClientSecret)
	if err != nil {
		return "", err
	}

	plaintext := pkcs7Pad([]byte(session.APIKey), aes.BlockSize)
	ciphertext := make([]byte, len(plaintext))
	mode := cipher.NewCBCEncrypter(block, webClientIV)
	mode.CryptBlocks(ciphertext, plaintext)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://api.vidio.com/livestreamings/%s/schedules?filter[date]=%s",
		channel.SiteID,
		date.Format("2006-01-02"),
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	apiKey, err := p.getAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("X-Secure-Level", "2")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from vidio", resp.StatusCode)
	}

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

	if len(response.Data) == 0 {
		return []domain.Program{}, nil
	}

	programs := make([]domain.Program, 0, len(response.Data))
	for _, item := range response.Data {
		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "vidio.com",
			Title:       item.Attributes.Title,
			Description: item.Attributes.Description,
			Image:       item.Attributes.ImageLandscapeURL,
			Lang:        channel.Lang,
		}

		if item.Attributes.StartTime != "" {
			if t, err := time.Parse(time.RFC3339, item.Attributes.StartTime); err == nil {
				p.Start = t.UTC()
			}
		}
		if item.Attributes.EndTime != "" {
			if t, err := time.Parse(time.RFC3339, item.Attributes.EndTime); err == nil {
				p.Stop = t.UTC()
			}
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type apiResponse struct {
	Data []scheduleItem `json:"data"`
}

type scheduleItem struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Attributes attributes `json:"attributes"`
}

type attributes struct {
	Title             string `json:"title"`
	StartTime         string `json:"start_time"`
	EndTime           string `json:"end_time"`
	Description       string `json:"description"`
	ImageLandscapeURL string `json:"image_landscape_url"`
}
