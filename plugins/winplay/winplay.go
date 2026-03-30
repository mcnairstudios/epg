package winplay

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

func (p *provider) Name() string        { return "winplay.co" }
func (p *provider) Description() string { return "Win+ Colombia streaming" }

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "winplay.co")
}

const graphQLQuery = `query getLivesEpg($page: Int = 1, $hours: Int, $ids: [String]) {
 getLives(ids: $ids) {
 _id
 logo
 name
 schedules(hours: $hours, page: {limit: 0, page: $page}) {
 _id
 name
 date_start
 date_end
 current
 match {
 matchDay
 __typename
 }
 show {
 _id
 title
 __typename
 }
 live {
 _id
 dvr
 type
 purchased
 __typename
 }
 __typename
 }
 __typename
 }
}`

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	reqBody := graphQLRequest{
		OperationName: "getLivesEpg",
		Variables: map[string]interface{}{
			"page":  1,
			"hours": 48,
		},
		Query: graphQLQuery,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, "https://next.platform.mediastre.am/graphql", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-client-id", "a084524ea449c15dfe5e75636fb55ce6a9d0d7601aac946daa")
	req.Header.Set("x-ott-language", "es")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from winplay", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(body, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var response graphQLResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	if response.Data == nil {
		return []domain.Program{}, nil
	}

	var schedules []scheduleItem
	for _, live := range response.Data.GetLives {
		if live.ID == channel.SiteID {
			schedules = live.Schedules
			break
		}
	}

	programs := make([]domain.Program, 0, len(schedules))
	for _, s := range schedules {
		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "winplay.co",
			Title:   s.Name,
			Lang:    channel.Lang,
		}

		if t, err := time.Parse(time.RFC3339Nano, s.DateStart); err == nil {
			p.Start = t.UTC()
		}
		if t, err := time.Parse(time.RFC3339Nano, s.DateEnd); err == nil {
			p.Stop = t.UTC()
		}

		programs = append(programs, p)
	}

	return programs, nil
}

type graphQLRequest struct {
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
	Query         string                 `json:"query"`
}

type graphQLResponse struct {
	Data *responseData `json:"data"`
}

type responseData struct {
	GetLives []liveChannel `json:"getLives"`
}

type liveChannel struct {
	ID        string         `json:"_id"`
	Name      string         `json:"name"`
	Schedules []scheduleItem `json:"schedules"`
}

type scheduleItem struct {
	Name      string `json:"name"`
	DateStart string `json:"date_start"`
	DateEnd   string `json:"date_end"`
}
