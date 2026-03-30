package tvcesoir_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvcesoir"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2023, 11, 24, 0, 0, 0, 0, time.UTC)
	programs, err := tvcesoir.Parse(data, domain.Channel{XmltvID: "Test", Lang: "xx", SiteID: "test"}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	expectedStart := time.Date(2023, 11, 24, 1, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	if p.Source != "tvcesoir.fr" {
		t.Fatalf("expected source 'tvcesoir.fr', got %q", p.Source)
	}

	if p.Title == "" {
		t.Fatal("expected title to be set")
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2023, 11, 24, 0, 0, 0, 0, time.UTC)
	programs, err := tvcesoir.Parse(data, domain.Channel{XmltvID: "Test", Lang: "xx", SiteID: "test"}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
