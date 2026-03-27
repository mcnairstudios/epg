package novacyprus

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	_ "embed"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

//go:embed channels.json
var channelsData []byte

func init() {
	tlsClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	plugin.Register(&provider{
		client: tlsClient,
	})
}

type provider struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (p *provider) Name() string        { return "novacyprus.com" }
func (p *provider) Description() string { return "Nova Cyprus TV" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "novacyprus.com")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	url := fmt.Sprintf(
		"https://www.novacyprus.com/api/v1/tvprogram/from/%s/to/%s",
		date.Format("20060102"), date.AddDate(0, 0, 1).Format("20060102"),
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
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

	programs := make([]domain.Program, 0)
	for _, item := range response.Nodes {
		if item.ChannelID != channel.SiteID {
			continue
		}

		var image string
		if len(item.MediaItems) > 0 {
			image = item.MediaItems[0].CdnURL
		}

		p := domain.Program{
			Channel:     channel.XmltvID,
			Source:      "novacyprus.com",
			Title:       item.Title,
			Description: item.Description,
			Image:       image,
			Lang:        channel.Lang,
		}

		if t, err := time.ParseInLocation("2006-01-02 15:04:05", item.DateTime, nicosiaLocation); err == nil {
			p.Start = t.UTC()
			dur, _ := strconv.Atoi(item.SlotDuration)
			p.Stop = t.Add(time.Duration(dur) * time.Minute).UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

var nicosiaLocation, _ = time.LoadLocation("Asia/Nicosia")

type apiResponse struct {
	Nodes []node `json:"nodes"`
}

type node struct {
	DateTime     string      `json:"datetime"`
	CID          string      `json:"cid"`
	ChannelID    string      `json:"ChannelId"`
	Title        string      `json:"title"`
	Description  string      `json:"description"`
	SlotDuration string      `json:"slotDuration"`
	MediaItems   []mediaItem `json:"mediaItems"`
}

type mediaItem struct {
	CdnURL string `json:"CdnUrl"`
}
