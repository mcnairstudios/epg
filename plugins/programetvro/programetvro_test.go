package programetvro

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

var testChannel = domain.Channel{
	ID:      "programetvro-pro-tv",
	XmltvID: "ProTV.ro",
	SiteID:  "pro-tv",
	Lang:    "ro",
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := Parse(data, testChannel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Ştirile Pro Tv" {
		t.Fatalf("expected title 'Ştirile Pro Tv', got %q", p.Title)
	}

	if p.Description != "În fiecare zi, cele mai importante evenimente, transmisiuni LIVE, analize, anchete şi reportaje sunt la Ştirile ProTV." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	expectedStart := time.Date(2021, 11, 7, 5, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2021, 11, 7, 7, 59, 59, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Category != "Ştiri" {
		t.Fatalf("unexpected category: %q", p.Category)
	}

	if p.Image != "https://www.programetv.ro/img/shows/84/54/stirile-pro-tv.png?key=Z2lfZnVial90cmFyZXZwLzAwLzAwLzA1LzE4MzgxMnktMTIwazE3MC1hLW40NTk4MW9zLmNhdA==" {
		t.Fatalf("unexpected image: %q", p.Image)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := Parse(data, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidContent(t *testing.T) {
	programs, err := Parse([]byte(`no pageData here`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseMultilinePageData(t *testing.T) {
	data := []byte(`<script>
var pageData = {
  "shows": [{"title": "Test Show", "start": "2021-11-07T07:00:00+02:00", "stop": "2021-11-07T08:00:00+02:00"}]
};
</script>`)

	programs, err := Parse(data, testChannel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}
	if programs[0].Title != "Test Show" {
		t.Fatalf("expected 'Test Show', got %q", programs[0].Title)
	}
}

func TestParseWhitespaceVariations(t *testing.T) {
	data := []byte(`var  pageData  =  {"shows":[{"title":"Whitespace","start":"2021-11-07T07:00:00+02:00","stop":"2021-11-07T08:00:00+02:00"}]};`)
	programs, err := Parse(data, testChannel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}
}

func TestParseEmptyShows(t *testing.T) {
	data := []byte(`var pageData = {"shows":[]};`)
	programs, err := Parse(data, testChannel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNullShows(t *testing.T) {
	data := []byte(`var pageData = {"shows":null};`)
	programs, err := Parse(data, testChannel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseObsFallback(t *testing.T) {
	data := []byte(`var pageData = {"shows":[{"title":"Obs Show","obs":"obs description","start":"2021-11-07T07:00:00+02:00","stop":"2021-11-07T08:00:00+02:00"}]};`)
	programs, err := Parse(data, testChannel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if programs[0].Description != "obs description" {
		t.Fatalf("expected obs fallback, got %q", programs[0].Description)
	}
}

type mockClient struct {
	responses []*http.Response
	errors    []error
	calls     atomic.Int32
}

func (m *mockClient) Do(_ *http.Request) (*http.Response, error) {
	idx := int(m.calls.Add(1)) - 1
	if idx < len(m.errors) && m.errors[idx] != nil {
		return nil, m.errors[idx]
	}
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return m.responses[len(m.responses)-1], nil
}

func makeResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{},
	}
}

func TestFetchSuccess(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	mc := &mockClient{
		responses: []*http.Response{makeResponse(200, string(data))},
	}

	p := &provider{client: mc}
	programs, err := p.Fetch(testChannel, time.Date(2021, 11, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}
}

func TestFetchNotFoundReturnsNil(t *testing.T) {
	mc := &mockClient{
		responses: []*http.Response{makeResponse(404, "not found")},
	}

	p := &provider{client: mc}
	programs, err := p.Fetch(testChannel, time.Date(2021, 11, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected nil error for 404, got: %v", err)
	}
	if programs != nil {
		t.Fatalf("expected nil programs for 404, got %d", len(programs))
	}
}

func TestFetchClientError(t *testing.T) {
	mc := &mockClient{
		responses: []*http.Response{makeResponse(403, "forbidden")},
	}

	p := &provider{client: mc}
	_, err := p.Fetch(testChannel, time.Date(2021, 11, 7, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("expected HTTP 403 in error, got: %v", err)
	}
}

func TestFetchRetryOn429(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	mc := &mockClient{
		responses: []*http.Response{
			makeResponse(429, "rate limited"),
			makeResponse(200, string(data)),
		},
	}

	p := &provider{client: mc}
	programs, err := p.Fetch(testChannel, time.Date(2021, 11, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected retry to succeed, got: %v", err)
	}
	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}
	if mc.calls.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", mc.calls.Load())
	}
}

func TestFetchRetryOnServerError(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	mc := &mockClient{
		responses: []*http.Response{
			makeResponse(502, "bad gateway"),
			makeResponse(503, "service unavailable"),
			makeResponse(200, string(data)),
		},
	}

	p := &provider{client: mc}
	programs, err := p.Fetch(testChannel, time.Date(2021, 11, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected retry to succeed, got: %v", err)
	}
	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}
	if mc.calls.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", mc.calls.Load())
	}
}

func TestFetchRetriesExhausted(t *testing.T) {
	mc := &mockClient{
		responses: []*http.Response{
			makeResponse(500, "error"),
			makeResponse(500, "error"),
			makeResponse(500, "error"),
			makeResponse(500, "error"),
		},
	}

	p := &provider{client: mc}
	_, err := p.Fetch(testChannel, time.Date(2021, 11, 7, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	if !strings.Contains(err.Error(), "after 3 retries") {
		t.Fatalf("expected retry count in error, got: %v", err)
	}
}

func TestFetchRetryOnTransportError(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	mc := &mockClient{
		responses: []*http.Response{
			nil,
			makeResponse(200, string(data)),
		},
		errors: []error{
			errors.New("connection reset"),
			nil,
		},
	}

	p := &provider{client: mc}
	programs, err := p.Fetch(testChannel, time.Date(2021, 11, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected retry to succeed after transport error, got: %v", err)
	}
	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}
}

func TestFetchSetsHeaders(t *testing.T) {
	var captured *http.Request
	mc := &mockClient{
		responses: []*http.Response{makeResponse(200, `var pageData = {"shows":[]};`)},
	}

	origDo := mc.Do
	_ = origDo

	headerCapture := &headerCapturingClient{
		inner:    mc,
		captured: &captured,
	}

	p := &provider{client: headerCapture}
	_, _ = p.Fetch(testChannel, time.Date(2021, 11, 7, 0, 0, 0, 0, time.UTC))

	if captured == nil {
		t.Fatal("no request captured")
	}
	if captured.Header.Get("User-Agent") == "" {
		t.Fatal("expected User-Agent header")
	}
	if captured.Header.Get("Accept-Language") == "" {
		t.Fatal("expected Accept-Language header")
	}
}

type headerCapturingClient struct {
	inner    httpClient
	captured **http.Request
}

func (h *headerCapturingClient) Do(req *http.Request) (*http.Response, error) {
	*h.captured = req
	return h.inner.Do(req)
}

func TestFetchDayMapping(t *testing.T) {
	tests := []struct {
		date    time.Time
		wantDay string
	}{
		{time.Date(2021, 11, 7, 0, 0, 0, 0, time.UTC), "duminica"},
		{time.Date(2021, 11, 8, 0, 0, 0, 0, time.UTC), "luni"},
		{time.Date(2021, 11, 9, 0, 0, 0, 0, time.UTC), "marti"},
		{time.Date(2021, 11, 10, 0, 0, 0, 0, time.UTC), "miercuri"},
		{time.Date(2021, 11, 11, 0, 0, 0, 0, time.UTC), "joi"},
		{time.Date(2021, 11, 12, 0, 0, 0, 0, time.UTC), "vineri"},
		{time.Date(2021, 11, 13, 0, 0, 0, 0, time.UTC), "sambata"},
	}

	for _, tt := range tests {
		t.Run(tt.wantDay, func(t *testing.T) {
			var captured *http.Request
			mc := &mockClient{
				responses: []*http.Response{makeResponse(200, `var pageData = {"shows":[]};`)},
			}
			headerCapture := &headerCapturingClient{
				inner:    mc,
				captured: &captured,
			}

			p := &provider{client: headerCapture}
			_, _ = p.Fetch(testChannel, tt.date)

			if captured == nil {
				t.Fatal("no request captured")
			}
			if !strings.Contains(captured.URL.Path, tt.wantDay) {
				t.Fatalf("expected %q in URL path, got %q", tt.wantDay, captured.URL.Path)
			}
		})
	}
}

func TestFlexIntVariations(t *testing.T) {
	tests := []struct {
		name string
		json string
		want int
	}{
		{"number", `var pageData = {"shows":[{"title":"T","season":3,"episode":5,"start":"2021-11-07T07:00:00+02:00","stop":"2021-11-07T08:00:00+02:00"}]};`, 3},
		{"string", `var pageData = {"shows":[{"title":"T","season":"3","episode":"5","start":"2021-11-07T07:00:00+02:00","stop":"2021-11-07T08:00:00+02:00"}]};`, 3},
		{"null", `var pageData = {"shows":[{"title":"T","season":null,"start":"2021-11-07T07:00:00+02:00","stop":"2021-11-07T08:00:00+02:00"}]};`, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			programs, err := Parse([]byte(tt.json), testChannel)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if len(programs) != 1 {
				t.Fatalf("expected 1 program, got %d", len(programs))
			}
			if programs[0].Season != tt.want {
				t.Fatalf("expected season %d, got %d", tt.want, programs[0].Season)
			}
		})
	}
}

func TestFlexStrSliceVariations(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantCat  string
	}{
		{"array", `var pageData = {"shows":[{"title":"T","categories":["A","B"],"start":"2021-11-07T07:00:00+02:00","stop":"2021-11-07T08:00:00+02:00"}]};`, "A, B"},
		{"string", `var pageData = {"shows":[{"title":"T","categories":"Drama","start":"2021-11-07T07:00:00+02:00","stop":"2021-11-07T08:00:00+02:00"}]};`, "Drama"},
		{"null", `var pageData = {"shows":[{"title":"T","categories":null,"start":"2021-11-07T07:00:00+02:00","stop":"2021-11-07T08:00:00+02:00"}]};`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			programs, err := Parse([]byte(tt.json), testChannel)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			if len(programs) != 1 {
				t.Fatalf("expected 1 program, got %d", len(programs))
			}
			if programs[0].Category != tt.wantCat {
				t.Fatalf("expected category %q, got %q", tt.wantCat, programs[0].Category)
			}
		})
	}
}
