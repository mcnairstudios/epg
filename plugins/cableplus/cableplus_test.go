package cableplus_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/cableplus"
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
		ID:      "cableplus.com.uy-2035",
		XmltvID: "APlusV.uy",
		SiteID:  "2035",
		Lang:    "es",
	}

	date := time.Date(2023, 2, 12, 0, 0, 0, 0, time.UTC)
	programs, err := cableplus.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	if len(programs) != 21 {
		t.Fatalf("expected 21 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Revista agropecuaria" {
		t.Fatalf("expected title 'Revista agropecuaria', got %q", p.Title)
	}

	if p.Image != "https://www.reportv.com.ar/buscador/img/Programas/2797844.jpg" {
		t.Fatalf("expected image URL, got %q", p.Image)
	}

	expectedStart := time.Date(2023, 2, 12, 9, 30, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{SiteID: "2035"}
	date := time.Date(2023, 2, 12, 0, 0, 0, 0, time.UTC)

	programs, err := cableplus.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse should not fail: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNoData(t *testing.T) {
	programs, err := cableplus.Parse([]byte(""), domain.Channel{}, time.Time{})
	if err != nil {
		t.Fatalf("parse should not fail: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
