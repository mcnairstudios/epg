package snrtma_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/snrtma"
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

	date := time.Date(2025, 1, 13, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "snrtma-4075",
		XmltvID: "Tamazight.ma",
		SiteID:  "4075",
		Lang:    "ar",
	}

	programs, err := snrtma.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 27 {
		t.Fatalf("expected 27 programs, got %d", len(programs))
	}

	p := programs[0]

	expectedStart := time.Date(2025, 1, 12, 23, 15, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 1, 12, 23, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "Tamazight.ma" {
		t.Fatalf("expected channel 'Tamazight.ma', got %q", p.Channel)
	}

	if p.Source != "snrt.ma" {
		t.Fatalf("expected source 'snrt.ma', got %q", p.Source)
	}

	last := programs[26]
	expectedLastStart := time.Date(2025, 1, 13, 21, 30, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := snrtma.Parse([]byte(""), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNoContent(t *testing.T) {
	content := `<!DOCTYPE html><html lang="ar" dir="rtl"><head></head><body></body></html>`
	programs, err := snrtma.Parse([]byte(content), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
