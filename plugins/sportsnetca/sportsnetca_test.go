package sportsnetca_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/sportsnetca"
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
		ID:      "watch.sportsnet.ca-24533",
		XmltvID: "SportsNetOntario.ca",
		SiteID:  "24533",
		Lang:    "en",
	}

	programs, err := sportsnetca.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Calgary Flames at Colorado Avalanche" {
		t.Fatalf("expected title 'Calgary Flames at Colorado Avalanche', got %q", p.Title)
	}

	if p.Description != "Johnny Gaudreau and the Flames pay a visit to the Avalanche. Calgary won 4-3 in overtime March 5." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	expectedStart := time.Date(2022, 3, 14, 0, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 3, 14, 2, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image == "" {
		t.Fatal("expected image to be set")
	}

	if p.Source != "watch.sportsnet.ca" {
		t.Fatalf("expected source 'watch.sportsnet.ca', got %q", p.Source)
	}
}

func TestParseEmptySchedule(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := sportsnetca.Parse(data, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := sportsnetca.Parse([]byte("not json"), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
