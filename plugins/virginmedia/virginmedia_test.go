package virginmedia_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/virginmedia"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseSegment(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content00.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "virginmedia-1958",
		XmltvID: "5ActionHD.uk",
		SiteID:  "1958",
		Lang:    "en",
	}

	programs, err := virginmedia.ParseSegment(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Police Interceptors" {
		t.Fatalf("expected title 'Police Interceptors', got %q", p.Title)
	}
	if p.Start.Format("2006-01-02T15:04:05.000Z") != "2024-12-14T00:00:00.000Z" {
		t.Fatalf("unexpected start: %v", p.Start)
	}
	if p.Stop.Format("2006-01-02T15:04:05.000Z") != "2024-12-14T01:05:00.000Z" {
		t.Fatalf("unexpected stop: %v", p.Stop)
	}
}

func TestParseProgram(t *testing.T) {
	data, err := os.ReadFile(testdataPath("program01.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "virginmedia-1958",
		XmltvID: "5ActionHD.uk",
		SiteID:  "1958",
		Lang:    "en",
	}

	p, err := virginmedia.ParseProgram(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if p.Title != "Police Interceptors" {
		t.Fatalf("expected title 'Police Interceptors', got %q", p.Title)
	}
	if p.Season != 16 {
		t.Fatalf("expected season 16, got %d", p.Season)
	}
	if p.Episode != 1 {
		t.Fatalf("expected episode 1, got %d", p.Episode)
	}
	if p.Category != "Reality" {
		t.Fatalf("expected category 'Reality', got %q", p.Category)
	}
	if p.Description == "" {
		t.Fatal("expected description to be set")
	}
}

func TestParseEmptyContent(t *testing.T) {
	channel := domain.Channel{SiteID: "1958", Lang: "en"}
	programs, err := virginmedia.ParseSegment([]byte(""), channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	channel := domain.Channel{SiteID: "1958", Lang: "en"}
	programs, err := virginmedia.ParseSegment([]byte(`not json`), channel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
