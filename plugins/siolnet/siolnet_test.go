package siolnet_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/siolnet"
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
		ID:      "tv-spored.siol.net-exodustv",
		XmltvID: "ExodusTV.si",
		SiteID:  "exodustv",
		Lang:    "sl",
	}

	date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	programs, err := siolnet.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 41 {
		t.Fatalf("expected 41 programs, got %d", len(programs))
	}

	first := programs[0]
	if first.Title != "Novice iz Svete de\u017eele" {
		t.Fatalf("expected title 'Novice iz Svete dežele', got %q", first.Title)
	}

	if first.Category != "informativni" {
		t.Fatalf("expected category 'informativni', got %q", first.Category)
	}

	expectedStart := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	if !first.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, first.Start)
	}

	expectedStop := time.Date(2025, 1, 15, 0, 30, 0, 0, time.UTC)
	if !first.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, first.Stop)
	}

	last := programs[40]
	if last.Title != "Sveta ma\u0161a" {
		t.Fatalf("expected last title 'Sveta maša', got %q", last.Title)
	}

	if last.Category != "ostalo" {
		t.Fatalf("expected last category 'ostalo', got %q", last.Category)
	}

	lastStart := time.Date(2025, 1, 15, 23, 0, 0, 0, time.UTC)
	if !last.Start.Equal(lastStart) {
		t.Fatalf("expected last start %v, got %v", lastStart, last.Start)
	}

	lastStop := time.Date(2025, 1, 15, 23, 45, 0, 0, time.UTC)
	if !last.Stop.Equal(lastStop) {
		t.Fatalf("expected last stop %v, got %v", lastStop, last.Stop)
	}
}

func TestParseEmptyHTML(t *testing.T) {
	programs, err := siolnet.Parse([]byte("<!DOCTYPE html><html><head></head><body></body></html>"), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse should not fail: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNoData(t *testing.T) {
	programs, err := siolnet.Parse([]byte(""), domain.Channel{}, time.Time{})
	if err != nil {
		t.Fatalf("parse should not fail: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
