package stod2_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/stod2"
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
		ID:      "syn",
		XmltvID: "Stod2.is@SD",
		SiteID:  "syn",
		Lang:    "is",
	}

	programs, err := stod2.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Bítiđ" {
		t.Fatalf("unexpected title: %q", p.Title)
	}

	if p.SubTitle != "Bítiđ" {
		t.Fatalf("unexpected subtitle: %q", p.SubTitle)
	}

	expectedStart := time.Date(2026, 3, 27, 6, 45, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	if p.Season != 2026 {
		t.Fatalf("expected season 2026, got %d", p.Season)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := stod2.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseEmptyArray(t *testing.T) {
	programs, err := stod2.Parse([]byte(`[]`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
