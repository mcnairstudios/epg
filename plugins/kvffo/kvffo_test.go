package kvffo_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/kvffo"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("example.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2021, 11, 21, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "kvffo-kvf",
		XmltvID: "KVFSjonvarp.fo",
		SiteID:  "#",
		Lang:    "fo",
	}

	programs, err := kvffo.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) < 3 {
		t.Fatalf("expected at least 3 programs, got %d", len(programs))
	}

	p := programs[2]
	if p.Title != "Letibygd 13" {
		t.Fatalf("expected title 'Letibygd 13', got %q", p.Title)
	}

	expectedStart := time.Date(2021, 11, 21, 18, 5, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2021, 11, 21, 18, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	date := time.Date(2021, 11, 21, 0, 0, 0, 0, time.UTC)
	programs, err := kvffo.Parse(
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
