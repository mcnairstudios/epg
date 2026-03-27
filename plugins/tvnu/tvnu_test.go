package tvnu_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvnu"
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
		ID:      "tvnu-3sat",
		XmltvID: "3sat.de",
		SiteID:  "3sat",
		Lang:    "sv",
	}

	programs, err := tvnu.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Natur im Garten" {
		t.Fatalf("expected title 'Natur im Garten', got %q", p.Title)
	}

	if p.Description != "Der Gartenbuchautor Karl Ploberger gibt in der Sendung Tipps und Tricks zur Gartenpflege." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	expectedStart := time.Date(2024, 12, 3, 11, 50, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2024, 12, 3, 12, 15, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Season != 29 {
		t.Fatalf("expected season 29, got %d", p.Season)
	}

	if p.Episode != 9 {
		t.Fatalf("expected episode 9, got %d", p.Episode)
	}

	if p.Category != "Konsument, Underhållning, Trädgård" {
		t.Fatalf("unexpected category: %q", p.Category)
	}

	if p.Channel != "3sat.de" {
		t.Fatalf("expected channel '3sat.de', got %q", p.Channel)
	}

	if p.Source != "tv.nu" {
		t.Fatalf("expected source 'tv.nu', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := tvnu.Parse(data, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := tvnu.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
