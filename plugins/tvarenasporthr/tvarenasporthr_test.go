package tvarenasporthr_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvarenasporthr"
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
		XmltvID: "ArenaSport1.hr",
		SiteID:  "01",
		Lang:    "hr",
	}

	date := time.Date(2024, 12, 7, 0, 0, 0, 0, time.UTC)
	programs, err := tvarenasporthr.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 15 {
		t.Fatalf("expected 15 programs, got %d", len(programs))
	}

	if programs[0].Title != "MAGAZIN" {
		t.Fatalf("expected title 'MAGAZIN', got %q", programs[0].Title)
	}

	expectedStart := time.Date(2024, 12, 7, 0, 0, 0, 0, time.UTC)
	if !programs[0].Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, programs[0].Start)
	}

	expectedStop := time.Date(2024, 12, 7, 0, 30, 0, 0, time.UTC)
	if !programs[0].Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, programs[0].Stop)
	}

	if programs[0].Description != "NBA ACTION" {
		t.Fatalf("expected description 'NBA ACTION', got %q", programs[0].Description)
	}

	if programs[0].Category != "Košarka" {
		t.Fatalf("expected category 'Košarka', got %q", programs[0].Category)
	}
}

func TestParseEmptyContent(t *testing.T) {
	channel := domain.Channel{SiteID: "01"}
	date := time.Date(2024, 12, 7, 0, 0, 0, 0, time.UTC)

	programs, err := tvarenasporthr.Parse([]byte(""), channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
