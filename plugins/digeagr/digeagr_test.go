package digeagr_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/digeagr"
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
		ID:      "digea.gr-1100",
		XmltvID: "AlphaTV.gr",
		SiteID:  "1100",
		Lang:    "el",
	}

	programs, err := digeagr.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 19 {
		t.Fatalf("expected 19 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "[K12] Το Ξεκαθάρισμα (A Score To Settle)" {
		t.Fatalf("expected title '[K12] Το Ξεκαθάρισμα (A Score To Settle)', got %q", p.Title)
	}

	expectedStart := time.Date(2025, 1, 16, 23, 30, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 1, 17, 1, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "AlphaTV.gr" {
		t.Fatalf("expected channel 'AlphaTV.gr', got %q", p.Channel)
	}

	if p.Source != "digea.gr" {
		t.Fatalf("expected source 'digea.gr', got %q", p.Source)
	}

	last := programs[18]
	if last.Title != "[K8] Βασικά Καλησπέρα Σας" {
		t.Fatalf("expected last title '[K8] Βασικά Καλησπέρα Σας', got %q", last.Title)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := digeagr.Parse([]byte(`[]`), domain.Channel{SiteID: "1100"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := digeagr.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
