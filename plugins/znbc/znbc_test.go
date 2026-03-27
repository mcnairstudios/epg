package znbc_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/znbc"
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

	date := time.Date(2021, 11, 25, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "znbc-tv1",
		XmltvID: "ZNBCTV1.zm",
		SiteID:  "tv1",
		Lang:    "en",
	}

	programs, err := znbc.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 3 {
		t.Fatalf("expected 3 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "MAIN NEWS – RPT" {
		t.Fatalf("expected title 'MAIN NEWS – RPT', got %q", p.Title)
	}

	expectedStart := time.Date(2021, 11, 24, 22, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2021, 11, 24, 23, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	p1 := programs[1]
	if p1.Title != "BORN & BRED – Rebroadcast (Tuesday Edition)" {
		t.Fatalf("expected title 'BORN & BRED – Rebroadcast (Tuesday Edition)', got %q", p1.Title)
	}

	expectedStart1 := time.Date(2021, 11, 24, 23, 0, 0, 0, time.UTC)
	if !p1.Start.Equal(expectedStart1) {
		t.Fatalf("expected start %v, got %v", expectedStart1, p1.Start)
	}

	expectedStop1 := time.Date(2021, 11, 25, 0, 0, 0, 0, time.UTC)
	if !p1.Stop.Equal(expectedStop1) {
		t.Fatalf("expected stop %v, got %v", expectedStop1, p1.Stop)
	}

	p2 := programs[2]
	if p2.Title != "DOCUMENTARY – DW" {
		t.Fatalf("expected title 'DOCUMENTARY – DW', got %q", p2.Title)
	}

	expectedStart2 := time.Date(2021, 11, 25, 0, 0, 0, 0, time.UTC)
	if !p2.Start.Equal(expectedStart2) {
		t.Fatalf("expected start %v, got %v", expectedStart2, p2.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	date := time.Date(2021, 11, 25, 0, 0, 0, 0, time.UTC)
	programs, err := znbc.Parse(
		[]byte("<!DOCTYPE html><html><head></head><body></body></html>"),
		domain.Channel{},
		date,
	)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
