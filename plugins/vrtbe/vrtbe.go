package vrtbe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
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

func (p *provider) Name() string        { return "vrt.be" }
func (p *provider) Description() string { return "VRT Belgian public broadcaster" }

const apiEndpoint = "https://www.vrt.be/vrtnu-api/graphql/public/v1"

const epgQuery = `
query EpgPage($pageId: ID!, $lazyItemCount: Int = 100) {
  page(id: $pageId) {
    ... on ElectronicProgramGuidePage {
      previous {
        ...epgListFragment
      }
      next {
        ...epgListFragment
      }
    }
  }
}

fragment epgListFragment on PaginatedTileList {
  listId
  paginatedItems(first: $lazyItemCount) {
    edges {
      cursor
      node {
        ...epgTileFragment
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}

fragment epgTileFragment on Tile {
  ... on ITile {
    title
    description
    primaryMeta {
      value
      shortValue
    }
    indexMeta {
      value
    }
    progress {
      durationInSeconds
    }
    status {
      accessibilityLabel
      text {
        small
        default
      }
    }
    image {
      templateUrl
    }
    action {
      ... on LinkAction {
        link
      }
    }
  }
}
`

func (p *provider) Channels() ([]domain.Channel, error) {
	return channels.Load(channelsData, "vrt.be")
}

func (p *provider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	body := graphqlRequest{
		Query: epgQuery,
		Variables: map[string]string{
			"pageId": fmt.Sprintf("/vrtmax/tv-gids/%s/%s/", channel.SiteID, date.Format("2006-01-02")),
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, apiEndpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Vrt-Client-Name", "WEB")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:136.0) Gecko/20100101 Firefox/136.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from vrtbe", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return Parse(data, channel)
}

func Parse(data []byte, channel domain.Channel) ([]domain.Program, error) {
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return []domain.Program{}, nil
	}

	if resp.Data == nil || resp.Data.Page == nil {
		return []domain.Program{}, nil
	}

	page := resp.Data.Page
	var edges []edge
	if page.Previous != nil && page.Previous.PaginatedItems != nil {
		edges = append(edges, page.Previous.PaginatedItems.Edges...)
	}
	if page.Next != nil && page.Next.PaginatedItems != nil {
		edges = append(edges, page.Next.PaginatedItems.Edges...)
	}

	var programs []domain.Program
	for i, e := range edges {
		if e.Node == nil || e.Node.Title == "" {
			continue
		}

		start := parseCursor(e.Cursor)
		if start.IsZero() {
			continue
		}

		var stop time.Time
		if i+1 < len(edges) {
			stop = parseCursor(edges[i+1].Cursor)
		} else {
			stop = parseFallbackStop(start, e.Node)
		}

		if stop.IsZero() || !stop.After(start) {
			continue
		}

		p := domain.Program{
			Channel: channel.XmltvID,
			Source:  "vrt.be",
			Title:   e.Node.Title,
			Lang:    channel.Lang,
			Start:   start,
			Stop:    stop,
			Season:  parseSeason(e.Node.PrimaryMeta),
			Episode: parseEpisode(e.Node.PrimaryMeta),
		}

		if e.Node.Description != "" {
			p.Description = e.Node.Description
		}
		if e.Node.Image != nil {
			p.Image = e.Node.Image.TemplateURL
		}

		programs = append(programs, p)
	}

	return programs, nil
}

var cursorRe = regexp.MustCompile(`^epg#[^#]+#(.+)$`)
var seasonRe = regexp.MustCompile(`^S(\d+)$`)
var episodeRe = regexp.MustCompile(`^Afl\.(\d+)$`)
var minRe = regexp.MustCompile(`(\d+)\s*min`)

func parseCursor(cursor string) time.Time {
	if cursor == "" {
		return time.Time{}
	}
	m := cursorRe.FindStringSubmatch(cursor)
	if m == nil {
		return time.Time{}
	}
	iso := m[1]
	t, err := time.Parse(time.RFC3339Nano, iso)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000Z", iso)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}

func parseFallbackStop(start time.Time, node *node) time.Time {
	if node.Progress != nil && node.Progress.DurationInSeconds > 0 {
		return start.Add(time.Duration(node.Progress.DurationInSeconds) * time.Second)
	}
	if node.Status != nil && node.Status.Text != nil && node.Status.Text.Small != "" {
		m := minRe.FindStringSubmatch(node.Status.Text.Small)
		if m != nil {
			mins, _ := strconv.Atoi(m[1])
			return start.Add(time.Duration(mins) * time.Minute)
		}
	}
	return time.Time{}
}

func parseSeason(metas []meta) int {
	for _, m := range metas {
		sm := seasonRe.FindStringSubmatch(m.ShortValue)
		if sm != nil {
			v, _ := strconv.Atoi(sm[1])
			return v
		}
	}
	return 0
}

func parseEpisode(metas []meta) int {
	for _, m := range metas {
		sv := strings.TrimPrefix(m.ShortValue, "Afl.")
		if sv != m.ShortValue {
			if em := episodeRe.FindStringSubmatch(m.ShortValue); em != nil {
				v, _ := strconv.Atoi(em[1])
				return v
			}
		}
	}
	return 0
}

type graphqlRequest struct {
	Query     string            `json:"query"`
	Variables map[string]string `json:"variables"`
}

type apiResponse struct {
	Data *apiData `json:"data"`
}

type apiData struct {
	Page *pageData `json:"page"`
}

type pageData struct {
	Previous *listData `json:"previous"`
	Next     *listData `json:"next"`
}

type listData struct {
	PaginatedItems *paginatedItems `json:"paginatedItems"`
}

type paginatedItems struct {
	Edges []edge `json:"edges"`
}

type edge struct {
	Cursor string `json:"cursor"`
	Node   *node  `json:"node"`
}

type node struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	PrimaryMeta []meta    `json:"primaryMeta"`
	Progress    *progress `json:"progress"`
	Status      *status   `json:"status"`
	Image       *image    `json:"image"`
}

type meta struct {
	Value      string `json:"value"`
	ShortValue string `json:"shortValue"`
}

type progress struct {
	DurationInSeconds int `json:"durationInSeconds"`
}

type status struct {
	Text *statusText `json:"text"`
}

type statusText struct {
	Small   string `json:"small"`
	Default string `json:"default"`
}

type image struct {
	TemplateURL string `json:"templateUrl"`
}
