package tvmusorhu_test

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvmusorhu"
)

type mockClient struct {
	statusCode int
	body       string
}

func (m *mockClient) Do(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(m.body)),
	}, nil
}

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
		ID:      "tvmusor-m4",
		XmltvID: "M4Sport.hu",
		SiteID:  "290",
		Lang:    "hu",
	}

	date := time.Date(2022, 11, 19, 0, 0, 0, 0, time.UTC)

	programs, err := tvmusorhu.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title == "" {
		t.Fatal("expected non-empty title")
	}

	if p.Source != "tvmusor.hu" {
		t.Fatalf("expected source 'tvmusor.hu', got %q", p.Source)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := tvmusorhu.Parse([]byte(`not json`), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("expected no error on invalid JSON, got %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseMissingBlock(t *testing.T) {
	programs, err := tvmusorhu.Parse(
		[]byte(`{"data":{"loadedBlocks":{}}}`),
		domain.Channel{SiteID: "999"},
		time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseDeduplicatesByB(t *testing.T) {
	data := []byte(`{"data":{"loadedBlocks":{"1_2022-01-01":[
		{"a":1,"b":100,"c":"","d":0,"e":1000,"f":2000,"h":"","j":"First","z":""},
		{"a":2,"b":100,"c":"","d":0,"e":3000,"f":4000,"h":"","j":"Duplicate B","z":""}
	]}}}`)

	channel := domain.Channel{SiteID: "1"}
	date := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

	programs, err := tvmusorhu.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 1 {
		t.Fatalf("expected 1 program after B dedup, got %d", len(programs))
	}
	if programs[0].Title != "First" {
		t.Fatalf("expected title 'First', got %q", programs[0].Title)
	}
}

func TestFetchNon200ReturnsEmpty(t *testing.T) {
	client := &mockClient{statusCode: 404, body: ""}
	p := tvmusorhu.NewProvider(client)

	channel := domain.Channel{SiteID: "999", XmltvID: "Test.hu"}
	date := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

	programs, err := p.Fetch(channel, date)
	if err != nil {
		t.Fatalf("expected no error for non-200, got %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs for non-200, got %d", len(programs))
	}
}

func TestFetchNon200Status500ReturnsEmpty(t *testing.T) {
	client := &mockClient{statusCode: 500, body: "internal server error"}
	p := tvmusorhu.NewProvider(client)

	channel := domain.Channel{SiteID: "123", XmltvID: "Test.hu"}
	date := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

	programs, err := p.Fetch(channel, date)
	if err != nil {
		t.Fatalf("expected no error for 500, got %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs for 500, got %d", len(programs))
	}
}

func TestFetchSuccess(t *testing.T) {
	body := `{"data":{"loadedBlocks":{"42_2022-01-01":[{"a":1,"b":2,"c":"desc","d":0,"e":1640995200000,"f":1640998800000,"h":"cat","j":"Show","z":"img.jpg"}]}}}`
	client := &mockClient{statusCode: 200, body: body}
	p := tvmusorhu.NewProvider(client)

	channel := domain.Channel{SiteID: "42", XmltvID: "Test.hu", Lang: "hu"}
	date := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

	programs, err := p.Fetch(channel, date)
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}
	if programs[0].Title != "Show" {
		t.Fatalf("expected title 'Show', got %q", programs[0].Title)
	}
}
