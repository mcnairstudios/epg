package mtsrs_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/mtsrs"
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
		ID:      "mtsrs-rts_1_hd",
		XmltvID: "RTS1HD.rs",
		SiteID:  "rts_1_hd",
		Lang:    "sr",
	}

	programs, err := mtsrs.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 31 {
		t.Fatalf("expected 31 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Jeloustoun" {
		t.Fatalf("expected title 'Jeloustoun', got %q", p.Title)
	}

	if p.Category != "Tv-serijali" {
		t.Fatalf("expected category 'Tv-serijali', got %q", p.Category)
	}

	expectedStart := time.Date(2025, 1, 22, 23, 25, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 1, 23, 0, 15, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image == "" {
		t.Fatal("expected image to be set")
	}

	last := programs[30]
	if last.Title != "Jeloustoun" {
		t.Fatalf("expected last title 'Jeloustoun', got %q", last.Title)
	}

	expectedLastStart := time.Date(2025, 1, 23, 23, 30, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}

	expectedLastStop := time.Date(2025, 1, 24, 0, 20, 0, 0, time.UTC)
	if !last.Stop.Equal(expectedLastStop) {
		t.Fatalf("expected last stop %v, got %v", expectedLastStop, last.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := mtsrs.Parse(data, domain.Channel{SiteID: "rts_1_hd"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := mtsrs.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatal("expected no error on invalid JSON")
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
