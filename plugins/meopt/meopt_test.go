package meopt_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/meopt"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "meopt-RTPM",
		XmltvID: "RTPMadeira.pt",
		SiteID:  "RTPM",
		Lang:    "pt",
	}

	programs, err := meopt.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Barman - Ep. 4" {
		t.Fatalf("expected title 'Barman - Ep. 4', got %q", p.Title)
	}

	expectedStart := time.Date(2026, 1, 22, 23, 40, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2026, 1, 23, 0, 4, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	expectedDesc := "'Barman' é uma série de comédia dramática sobre um jovem comediante que começa a trabalhar como Barman porque precisa de arranjar dinheiro depressa, pelo caminho é obrigado a lidar com a vida noturna e conciliar duas realidades diferentes."
	if p.Description != expectedDesc {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	expectedImage := "https://cdn-er-images.online.meo.pt/eemstb/ImageHandler.ashx?chCallLetter=RTPM&progTitle=Barman+-+Ep.+4&profile=16_9&profileFallback=false&noFallback=true&appSource=PC_CHROME_PWA&width=1920&csf"
	if p.Image != expectedImage {
		t.Fatalf("expected image %q, got %q", expectedImage, p.Image)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := meopt.Parse([]byte(""), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if programs != nil {
		t.Fatalf("expected nil, got %d programs", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := meopt.Parse([]byte("not json"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if programs != nil {
		t.Fatalf("expected nil, got %d programs", len(programs))
	}
}

func TestParseAPIErrorStatus(t *testing.T) {
	response := map[string]interface{}{
		"Status":     "Error",
		"StatusCode": 500,
		"Result":     []interface{}{},
	}
	data, _ := json.Marshal(response)

	programs, err := meopt.Parse(data, domain.Channel{SiteID: "TEST"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if programs != nil {
		t.Fatalf("expected nil for API error status, got %d programs", len(programs))
	}
}

func TestParseSkipsBadStartDate(t *testing.T) {
	response := map[string]interface{}{
		"Status":     "OK",
		"StatusCode": 200,
		"Result": []interface{}{
			map[string]interface{}{
				"CallLetter": "TEST",
				"StartDate":  "bad-date",
				"EndDate":    "2026-01-22T10:00:00",
				"Title":      "Good Title",
			},
			map[string]interface{}{
				"CallLetter": "TEST",
				"StartDate":  "2026-01-22T10:00:00",
				"EndDate":    "2026-01-22T11:00:00",
				"Title":      "Valid Program",
			},
		},
	}
	data, _ := json.Marshal(response)

	ch := domain.Channel{XmltvID: "Test.pt", SiteID: "TEST", Lang: "pt"}
	programs, err := meopt.Parse(data, ch)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 1 {
		t.Fatalf("expected 1 program (skipping bad date), got %d", len(programs))
	}
	if programs[0].Title != "Valid Program" {
		t.Fatalf("expected 'Valid Program', got %q", programs[0].Title)
	}
}

func TestParseSkipsBadEndDate(t *testing.T) {
	response := map[string]interface{}{
		"Status":     "OK",
		"StatusCode": 200,
		"Result": []interface{}{
			map[string]interface{}{
				"CallLetter": "TEST",
				"StartDate":  "2026-01-22T10:00:00",
				"EndDate":    "not-a-date",
				"Title":      "Bad End",
			},
		},
	}
	data, _ := json.Marshal(response)

	ch := domain.Channel{XmltvID: "Test.pt", SiteID: "TEST", Lang: "pt"}
	programs, err := meopt.Parse(data, ch)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyTitle(t *testing.T) {
	response := map[string]interface{}{
		"Status":     "OK",
		"StatusCode": 200,
		"Result": []interface{}{
			map[string]interface{}{
				"CallLetter": "TEST",
				"StartDate":  "2026-01-22T10:00:00",
				"EndDate":    "2026-01-22T11:00:00",
				"Title":      "",
			},
		},
	}
	data, _ := json.Marshal(response)

	ch := domain.Channel{XmltvID: "Test.pt", SiteID: "TEST", Lang: "pt"}
	programs, err := meopt.Parse(data, ch)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}
	if programs[0].Title != "Sem título" {
		t.Fatalf("expected 'Sem título', got %q", programs[0].Title)
	}
	if programs[0].Image != "" {
		t.Fatalf("expected no image for empty title, got %q", programs[0].Image)
	}
}

type mockClient struct {
	status int
	body   []byte
	err    error
	req    *http.Request
}

func (m *mockClient) Do(req *http.Request) (*http.Response, error) {
	m.req = req
	if m.err != nil {
		return nil, m.err
	}
	return &http.Response{
		StatusCode: m.status,
		Body:       io.NopCloser(bytes.NewReader(m.body)),
	}, nil
}

func TestFetchSetsHeaders(t *testing.T) {
	response := map[string]interface{}{
		"Status":     "OK",
		"StatusCode": 200,
		"Result":     []interface{}{},
	}
	body, _ := json.Marshal(response)

	mock := &mockClient{status: 200, body: body}
	p := meopt.NewProvider(mock)

	ch := domain.Channel{SiteID: "RTP1", XmltvID: "RTP1.pt", Lang: "pt"}
	_, _ = p.Fetch(ch, time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC))

	if mock.req == nil {
		t.Fatal("expected request to be made")
	}

	expected := map[string]string{
		"Origin":     "https://www.meo.pt",
		"Referer":    "https://www.meo.pt/",
		"User-Agent": "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Mobile Safari/537.36",
	}
	for key, val := range expected {
		if got := mock.req.Header.Get(key); got != val {
			t.Fatalf("header %s: expected %q, got %q", key, val, got)
		}
	}
}

func TestFetchHTTPErrorReturnsNil(t *testing.T) {
	mock := &mockClient{status: 404, body: []byte("not found")}
	p := meopt.NewProvider(mock)

	ch := domain.Channel{SiteID: "BADCH", XmltvID: "Bad.pt", Lang: "pt"}
	programs, err := p.Fetch(ch, time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if programs != nil {
		t.Fatalf("expected nil for 404, got %d programs", len(programs))
	}
}

func TestFetchHTTP500ReturnsNil(t *testing.T) {
	mock := &mockClient{status: 500, body: []byte("server error")}
	p := meopt.NewProvider(mock)

	ch := domain.Channel{SiteID: "ERR", XmltvID: "Err.pt", Lang: "pt"}
	programs, err := p.Fetch(ch, time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if programs != nil {
		t.Fatalf("expected nil for 500, got %d programs", len(programs))
	}
}
