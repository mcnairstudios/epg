package mncvision_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/mncvision"
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
		ID:      "mncvision-154",
		XmltvID: "AXN.id",
		SiteID:  "154",
		Lang:    "id",
	}

	date := time.Date(2023, 11, 19, 0, 0, 0, 0, time.UTC)
	programs, err := mncvision.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Blue Bloods S13, Ep 19" {
		t.Fatalf("expected title %q, got %q", "Blue Bloods S13, Ep 19", p.Title)
	}

	if p.Episode != 19 {
		t.Fatalf("expected episode 19, got %d", p.Episode)
	}

	if p.Start.UTC().Format("2006-01-02T15:04:05.000Z") != "2023-11-18T17:15:00.000Z" {
		t.Fatalf("expected start 2023-11-18T17:15:00.000Z, got %v", p.Start.UTC().Format("2006-01-02T15:04:05.000Z"))
	}

	if p.Stop.UTC().Format("2006-01-02T15:04:05.000Z") != "2023-11-18T18:05:00.000Z" {
		t.Fatalf("expected stop 2023-11-18T18:05:00.000Z, got %v", p.Stop.UTC().Format("2006-01-02T15:04:05.000Z"))
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2023, 11, 19, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{SiteID: "154", XmltvID: "AXN.id", Lang: "id"}
	programs, err := mncvision.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if programs != nil && len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
