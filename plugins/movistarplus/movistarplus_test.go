package movistarplus_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/movistarplus"
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
		ID:      "movistarplus-sexta",
		XmltvID: "LaSexta.es",
		SiteID:  "sexta",
		Lang:    "es",
	}

	date := time.Date(2025, 5, 30, 0, 0, 0, 0, time.UTC)
	programs, err := movistarplus.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 23 {
		t.Fatalf("expected 23 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Venta Prime" {
		t.Fatalf("expected title %q, got %q", "Venta Prime", p.Title)
	}

	if p.Start.UTC().Format("2006-01-02T15:04:05.000Z") != "2025-05-30T03:15:00.000Z" {
		t.Fatalf("expected start 2025-05-30T03:15:00.000Z, got %v", p.Start.UTC().Format("2006-01-02T15:04:05.000Z"))
	}

	if p.Stop.UTC().Format("2006-01-02T15:04:05.000Z") != "2025-05-30T04:25:00.000Z" {
		t.Fatalf("expected stop 2025-05-30T04:25:00.000Z, got %v", p.Stop.UTC().Format("2006-01-02T15:04:05.000Z"))
	}
}

func TestParseProgramDescription(t *testing.T) {
	data, err := os.ReadFile(testdataPath("program1.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	desc := movistarplus.ParseProgramDescription(data)
	if desc != "Espacio de televenta." {
		t.Fatalf("expected description %q, got %q", "Espacio de televenta.", desc)
	}
}

func TestParseEmptyContent(t *testing.T) {
	date := time.Date(2025, 5, 30, 0, 0, 0, 0, time.UTC)
	programs, err := movistarplus.Parse([]byte("[]"), domain.Channel{SiteID: "sexta"}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if programs != nil && len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
