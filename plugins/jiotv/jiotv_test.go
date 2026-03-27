package jiotv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/jiotv"
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
		ID:      "jiotv-1",
		XmltvID: "JioTV.in",
		SiteID:  "1",
		Lang:    "en",
	}

	programs, err := jiotv.Parse(data, channel)
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

	if p.Source != "jiotv.com" {
		t.Fatalf("expected source 'jiotv.com', got %q", p.Source)
	}

	if p.Start.IsZero() {
		t.Fatal("expected non-zero start time")
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := jiotv.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseEmptyEPG(t *testing.T) {
	programs, err := jiotv.Parse([]byte(`{"epg":[]}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
