package clickthecity_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/clickthecity"
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

	date := time.Date(2023, 6, 12, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "clickthecity-5",
		XmltvID: "TV5.ph",
		SiteID:  "5",
		Lang:    "en",
	}

	programs, err := clickthecity.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 20 {
		t.Fatalf("expected 20 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Word Of God" {
		t.Fatalf("expected title 'Word Of God', got %q", p.Title)
	}

	expectedStart := time.Date(2023, 6, 11, 21, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 6, 11, 22, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	last := programs[19]

	if last.Title != "La Suerte De Loli" {
		t.Fatalf("expected last title 'La Suerte De Loli', got %q", last.Title)
	}

	expectedLastStart := time.Date(2023, 6, 12, 15, 30, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}

	expectedLastStop := time.Date(2023, 6, 12, 16, 0, 0, 0, time.UTC)
	if !last.Stop.Equal(expectedLastStop) {
		t.Fatalf("expected last stop %v, got %v", expectedLastStop, last.Stop)
	}

	if p.Channel != "TV5.ph" {
		t.Fatalf("expected channel 'TV5.ph', got %q", p.Channel)
	}

	if p.Source != "clickthecity.com" {
		t.Fatalf("expected source 'clickthecity.com', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data := []byte(`<!DOCTYPE html><html class="html" lang="en-US"><head></head><body></body></html>`)
	programs, err := clickthecity.Parse(data, domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
