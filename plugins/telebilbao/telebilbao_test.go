package telebilbao_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/telebilbao"
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
		XmltvID: "TeleBilbao.es",
		SiteID:  "#",
		Lang:    "es",
	}

	date := time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC)
	programs, err := telebilbao.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 50 {
		t.Fatalf("expected 50 programs, got %d", len(programs))
	}

	expectedStart := time.Date(2025, 1, 16, 6, 0, 0, 0, time.UTC)
	if !programs[0].Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, programs[0].Start)
	}

	if programs[0].Title != "BAI HORIXE" {
		t.Fatalf("expected title 'BAI HORIXE', got %q", programs[0].Title)
	}

	expectedLastStart := time.Date(2025, 1, 17, 7, 30, 0, 0, time.UTC)
	if !programs[49].Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, programs[49].Start)
	}

	if programs[49].Title != "LA KAPITAL" {
		t.Fatalf("expected last title 'LA KAPITAL', got %q", programs[49].Title)
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{SiteID: "#"}
	date := time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC)

	programs, err := telebilbao.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := telebilbao.Parse([]byte(""), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
