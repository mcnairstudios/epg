package tataplay_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tataplay"
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
		ID:      "tataplay-1001",
		XmltvID: "StarPlus.in",
		SiteID:  "1001",
		Lang:    "en",
	}

	programs, err := tataplay.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Yeh Rishta Kya Kehlata Hai" {
		t.Fatalf("expected title %q, got %q", "Yeh Rishta Kya Kehlata Hai", p.Title)
	}

	if p.Description != "The story of the Rajshri family and their journey through life." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	if p.Category != "Drama" {
		t.Fatalf("expected category %q, got %q", "Drama", p.Category)
	}

	expectedStart := time.Date(2025, 6, 9, 18, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 6, 9, 18, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	p2 := programs[1]
	if p2.Title != "Anupamaa" {
		t.Fatalf("expected title %q, got %q", "Anupamaa", p2.Title)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := tataplay.Parse([]byte(`{"data":{"epg":[]}}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := tataplay.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if programs != nil && len(programs) != 0 {
		t.Fatalf("expected nil/empty programs, got %d", len(programs))
	}
}
