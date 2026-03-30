package zap2it

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
)

func fetchTestdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

var testChannel = domain.Channel{
	ID:      "zap2it.com-X/USA-NY31695-DEFAULT/NY31695/USA/13302/49141",
	XmltvID: "SpectrumNews1.us",
	SiteID:  "X/USA-NY31695-DEFAULT/NY31695/USA/13302/49141",
	Lang:    "en",
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

func noSleep(_ time.Duration) {}

func TestFetchSuccess(t *testing.T) {
	data, err := os.ReadFile(fetchTestdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	mc := &mockClient{
		responses: []*http.Response{makeResponse(200, string(data))},
	}

	p := &provider{client: mc, sleep: noSleep}
	programs, err := p.Fetch(testChannel, time.Date(2025, 2, 6, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}
}

func TestFetchInvalidSiteID(t *testing.T) {
	ch := domain.Channel{SiteID: "bad-format"}
	p := &provider{client: &mockClient{}, sleep: noSleep}
	_, err := p.Fetch(ch, time.Now())
	if err == nil {
		t.Fatal("expected error for invalid site_id")
	}
}

func TestFetchNon200ReturnsError(t *testing.T) {
	mc := &mockClient{
		responses: []*http.Response{makeResponse(403, "forbidden")},
	}

	p := &provider{client: mc, sleep: noSleep}
	_, err := p.Fetch(testChannel, time.Now())
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
}

func TestFetch429RetriesThenReturnsNil(t *testing.T) {
	mc := &mockClient{
		responses: []*http.Response{
			makeResponse(429, ""),
			makeResponse(429, ""),
			makeResponse(429, ""),
			makeResponse(429, ""),
		},
	}

	p := &provider{client: mc, sleep: noSleep}
	programs, err := p.Fetch(testChannel, time.Now())
	if err != nil {
		t.Fatalf("expected nil error after 429 retries, got: %v", err)
	}
	if programs != nil {
		t.Fatalf("expected nil programs after 429 retries, got %d", len(programs))
	}
	if mc.calls.Load() != 4 {
		t.Fatalf("expected 4 attempts (1 + 3 retries), got %d", mc.calls.Load())
	}
}

func TestFetch429RetryThenSuccess(t *testing.T) {
	data, err := os.ReadFile(fetchTestdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	mc := &mockClient{
		responses: []*http.Response{
			makeResponse(429, ""),
			makeResponse(200, string(data)),
		},
	}

	p := &provider{client: mc, sleep: noSleep}
	programs, err := p.Fetch(testChannel, time.Date(2025, 2, 6, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}
	if mc.calls.Load() != 2 {
		t.Fatalf("expected 2 attempts, got %d", mc.calls.Load())
	}
}

func TestFetch500RetriesThenReturnsNil(t *testing.T) {
	mc := &mockClient{
		responses: []*http.Response{
			makeResponse(500, ""),
			makeResponse(502, ""),
			makeResponse(503, ""),
			makeResponse(500, ""),
		},
	}

	p := &provider{client: mc, sleep: noSleep}
	programs, err := p.Fetch(testChannel, time.Now())
	if err != nil {
		t.Fatalf("expected nil error after 5xx retries, got: %v", err)
	}
	if programs != nil {
		t.Fatalf("expected nil programs after 5xx retries, got %d", len(programs))
	}
	if mc.calls.Load() != 4 {
		t.Fatalf("expected 4 attempts, got %d", mc.calls.Load())
	}
}

func TestFetchNetworkErrorRetriesThenFails(t *testing.T) {
	mc := &mockClient{
		errors: []error{
			errors.New("connection refused"),
			errors.New("connection refused"),
			errors.New("connection refused"),
			errors.New("connection refused"),
		},
		responses: []*http.Response{nil, nil, nil, nil},
	}

	p := &provider{client: mc, sleep: noSleep}
	_, err := p.Fetch(testChannel, time.Now())
	if err == nil {
		t.Fatal("expected error after network failures")
	}
	if mc.calls.Load() != 4 {
		t.Fatalf("expected 4 attempts, got %d", mc.calls.Load())
	}
}

func TestFetchNetworkErrorRetryThenSuccess(t *testing.T) {
	data, err := os.ReadFile(fetchTestdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	mc := &mockClient{
		errors: []error{
			errors.New("connection refused"),
			nil,
		},
		responses: []*http.Response{
			nil,
			makeResponse(200, string(data)),
		},
	}

	p := &provider{client: mc, sleep: noSleep}
	programs, err := p.Fetch(testChannel, time.Date(2025, 2, 6, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}
}
