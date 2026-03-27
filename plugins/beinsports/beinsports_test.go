package beinsports_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/beinsports"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "beinsports-usa",
		XmltvID: "beINSportsUSA.us",
		SiteID:  "C244C48D-3B54-406A-94C9-D63B16318267",
		Lang:    "en",
	}

	programs, err := beinsports.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Tokyo Day 5 QF 2" {
		t.Fatalf("expected title 'Tokyo Day 5 QF 2', got %q", p.Title)
	}
	if p.Description != "Exclusive coverage of the 2023 ATP Tour on beIN SPORTS" {
		t.Fatalf("unexpected description: %q", p.Description)
	}
	if p.Start.Format("2006-01-02T15:04:05.000Z") != "2023-10-20T23:30:00.000Z" {
		t.Fatalf("unexpected start: %v", p.Start)
	}
	if p.Stop.Format("2006-01-02T15:04:05.000Z") != "2023-10-21T01:00:00.000Z" {
		t.Fatalf("unexpected stop: %v", p.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := beinsports.Parse([]byte(`[]`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := beinsports.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
