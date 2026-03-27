package rotana_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/rotana"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseSchedule(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_en.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2024, 11, 26, 0, 0, 0, 0, time.UTC)
	items := rotana.ParseSchedule(data, date)

	if len(items) != 12 {
		t.Fatalf("expected 12 items, got %d", len(items))
	}

	last := items[11]
	if last.Title != "Khiyana Mashroua" {
		t.Fatalf("expected last title 'Khiyana Mashroua', got %q", last.Title)
	}

	expectedLastStart := time.Date(2024, 11, 26, 20, 0, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}

	expectedLastStop := time.Date(2024, 11, 26, 22, 0, 0, 0, time.UTC)
	if !last.Stop.Equal(expectedLastStop) {
		t.Fatalf("expected last stop %v, got %v", expectedLastStop, last.Stop)
	}
}

func TestParseProgramDetail(t *testing.T) {
	data, err := os.ReadFile(testdataPath("program_en.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	detail := rotana.ParseProgramDetail(data)
	if detail == nil {
		t.Fatal("expected detail, got nil")
	}

	if detail.Category != "Movie" {
		t.Fatalf("expected category 'Movie', got %q", detail.Category)
	}

	if detail.Image == "" {
		t.Fatal("expected image to be set")
	}
}

func TestParseProgramDetailArabic(t *testing.T) {
	data, err := os.ReadFile(testdataPath("program_ar.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	detail := rotana.ParseProgramDetail(data)
	if detail == nil {
		t.Fatal("expected detail, got nil")
	}

	if detail.Image == "" {
		t.Fatal("expected image to be set")
	}
}

func TestParseEmptySchedule(t *testing.T) {
	items := rotana.ParseSchedule([]byte("<!DOCTYPE html><html><head></head><body></body></html>"), time.Now())
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestToPrograms(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_en.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2024, 11, 26, 0, 0, 0, 0, time.UTC)
	items := rotana.ParseSchedule(data, date)

	channel := domain.Channel{
		XmltvID: "RotanaCinemaMasr.sa",
		SiteID:  "439",
		Lang:    "en",
	}

	programs := make([]domain.Program, 0, len(items))
	for _, item := range items {
		programs = append(programs, domain.Program{
			Channel: channel.XmltvID,
			Source:  "rotana.net",
			Title:   item.Title,
			Lang:    channel.Lang,
			Start:   item.Start,
			Stop:    item.Stop,
		})
	}

	if len(programs) != 12 {
		t.Fatalf("expected 12 programs, got %d", len(programs))
	}

	if programs[0].Source != "rotana.net" {
		t.Fatalf("expected source 'rotana.net', got %q", programs[0].Source)
	}
}
