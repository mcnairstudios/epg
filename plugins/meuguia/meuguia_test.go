package meuguia_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/meuguia"
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

	date := time.Date(2023, 11, 21, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		SiteID:  "AXN",
		XmltvID: "AXN.id",
		Lang:    "pt",
	}

	programs, err := meuguia.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 3 {
		t.Fatalf("expected 3 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Hawaii Five-0 : T10 EP4 - Tiny Is the Flower, Yet It Scents the Grasses Around It" {
		t.Fatalf("unexpected title: %q", p.Title)
	}
	if p.Season != 10 {
		t.Fatalf("expected season 10, got %d", p.Season)
	}
	if p.Episode != 4 {
		t.Fatalf("expected episode 4, got %d", p.Episode)
	}

	expectedStart := time.Date(2023, 11, 21, 21, 20, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 11, 21, 22, 15, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	last := programs[2]
	if last.Season != 5 {
		t.Fatalf("expected season 5, got %d", last.Season)
	}
	if last.Episode != 15 {
		t.Fatalf("expected episode 15, got %d", last.Episode)
	}

	expectedStart = time.Date(2023, 11, 21, 23, 10, 0, 0, time.UTC)
	if !last.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, last.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := meuguia.Parse(
		[]byte("<!DOCTYPE html><html><head></head><body></body></html>"),
		domain.Channel{},
		time.Date(2023, 11, 21, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
