package thesportplus_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/thesportplus"
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

	channel := domain.Channel{
		ID:      "thesportplus.com-usa",
		XmltvID: "SportPlusUSA.us",
		SiteID:  "usa",
		Lang:    "en",
	}

	date := time.Date(2025, 1, 19, 0, 0, 0, 0, time.UTC)
	programs, err := thesportplus.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 13 {
		t.Fatalf("expected 13 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "ASTERAS vs ATROMITOS" {
		t.Fatalf("expected title 'ASTERAS vs ATROMITOS', got %q", p.Title)
	}
	if p.Description != "Super League Season 24-25 MD 4" {
		t.Fatalf("expected description 'Super League Season 24-25 MD 4', got %q", p.Description)
	}
	expectedStart := time.Date(2025, 1, 19, 6, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
	expectedStop := time.Date(2025, 1, 19, 8, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	last := programs[12]
	if last.Title != "SPORTSHOW" {
		t.Fatalf("expected last title 'SPORTSHOW', got %q", last.Title)
	}
	expectedLastStart := time.Date(2025, 1, 20, 4, 0, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}
	expectedLastStop := time.Date(2025, 1, 20, 5, 0, 0, 0, time.UTC)
	if !last.Stop.Equal(expectedLastStop) {
		t.Fatalf("expected last stop %v, got %v", expectedLastStop, last.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "thesportplus.com-usa",
		XmltvID: "SportPlusUSA.us",
		SiteID:  "usa",
		Lang:    "en",
	}

	date := time.Date(2025, 1, 19, 0, 0, 0, 0, time.UTC)
	programs, err := thesportplus.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidHTML(t *testing.T) {
	programs, err := thesportplus.Parse([]byte(""), domain.Channel{SiteID: "usa"}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
