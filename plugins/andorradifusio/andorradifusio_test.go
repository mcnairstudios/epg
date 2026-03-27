package andorradifusio_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/andorradifusio"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

var channel = domain.Channel{
	ID:      "andorradifusio.ad-atv",
	XmltvID: "AndorraTV.ad@SD",
	SiteID:  "atv",
	Lang:    "ca",
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	programs, err := andorradifusio.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 5 {
		t.Fatalf("expected 5 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Club Piolet" {
		t.Fatalf("expected title 'Club Piolet', got %q", p.Title)
	}

	expectedStart := time.Date(2026, 4, 1, 7, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2026, 4, 1, 7, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "AndorraTV.ad@SD" {
		t.Fatalf("expected channel 'AndorraTV.ad@SD', got %q", p.Channel)
	}

	if p.Source != "andorradifusio.ad" {
		t.Fatalf("expected source 'andorradifusio.ad', got %q", p.Source)
	}

	if p.Lang != "ca" {
		t.Fatalf("expected lang 'ca', got %q", p.Lang)
	}

	last := programs[4]
	if last.Title != "Àrea Andorra" {
		t.Fatalf("expected last title 'Àrea Andorra', got %q", last.Title)
	}

	expectedLastStart := time.Date(2026, 4, 1, 21, 0, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}
}

func TestParseFiltersOutOtherChannels(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	programs, err := andorradifusio.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	for _, p := range programs {
		if p.Channel != "AndorraTV.ad@SD" {
			t.Fatalf("found program for wrong channel: %q", p.Channel)
		}
	}
}

func TestParseFiltersOutOtherDates(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	programs, err := andorradifusio.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program for March 31, got %d", len(programs))
	}

	if programs[0].Title != "Yesterday Show" {
		t.Fatalf("expected title 'Yesterday Show', got %q", programs[0].Title)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := andorradifusio.Parse([]byte("<!DOCTYPE html><html><head></head><body></body></html>"), channel, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyData(t *testing.T) {
	programs, err := andorradifusio.Parse([]byte(""), channel, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
