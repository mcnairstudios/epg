package stod2_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/stod2"
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
		SiteID:  "syn",
		XmltvID: "Stod2.is",
		Lang:    "is",
	}

	programs, err := stod2.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Fréttir" {
		t.Fatalf("expected title %q, got %q", "Fréttir", p.Title)
	}
	if p.SubTitle != "Morgunfréttir" {
		t.Fatalf("expected subtitle %q, got %q", "Morgunfréttir", p.SubTitle)
	}
	if p.Description != "Fréttir dagsins" {
		t.Fatalf("expected description %q, got %q", "Fréttir dagsins", p.Description)
	}
	if p.Season != 1 {
		t.Fatalf("expected season 1, got %d", p.Season)
	}
	if p.Episode != 5 {
		t.Fatalf("expected episode 5, got %d", p.Episode)
	}

	expectedStart := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2024, 1, 1, 10, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	p2 := programs[1]
	if p2.Season != 0 {
		t.Fatalf("expected season 0, got %d", p2.Season)
	}
	if p2.Episode != 0 {
		t.Fatalf("expected episode 0, got %d", p2.Episode)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := stod2.Parse([]byte("not json"), domain.Channel{})
	if err != nil {
		t.Fatalf("expected nil error for invalid JSON, got: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyArray(t *testing.T) {
	programs, err := stod2.Parse([]byte("[]"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
