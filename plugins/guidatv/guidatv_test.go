package guidatv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/guidatv"
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

	date := time.Date(2023, 11, 24, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "guidatv-rai1",
		XmltvID: "Rai1.it",
		SiteID:  "826573/rai-1",
		Lang:    "it",
	}

	programs, err := guidatv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Viva Rai2!" {
		t.Fatalf("expected title 'Viva Rai2!', got %q", p.Title)
	}
	if p.Start.Format("2006-01-02T15:04:05.000Z") != "2023-11-24T00:10:00.000Z" {
		t.Fatalf("unexpected start: %v", p.Start)
	}
	if p.Stop.Format("2006-01-02T15:04:05.000Z") != "2023-11-24T01:05:00.000Z" {
		t.Fatalf("unexpected stop: %v", p.Stop)
	}

	if len(programs) < 31 {
		t.Fatalf("expected at least 31 programs, got %d", len(programs))
	}

	last := programs[30]
	if last.Title != "TV 7" {
		t.Fatalf("expected title 'TV 7', got %q", last.Title)
	}
	if last.Start.Format("2006-01-02T15:04:05.000Z") != "2023-11-24T23:00:00.000Z" {
		t.Fatalf("unexpected start for last: %v", last.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	date := time.Date(2023, 11, 24, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{SiteID: "826573/rai-1", Lang: "it"}
	programs, err := guidatv.Parse(
		[]byte("<!DOCTYPE html><html><head></head><body></body></html>"),
		channel,
		date,
	)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidHTML(t *testing.T) {
	date := time.Date(2023, 11, 24, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{SiteID: "826573/rai-1", Lang: "it"}
	programs, err := guidatv.Parse([]byte(""), channel, date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
