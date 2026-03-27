package zap2it_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/zap2it"
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
		SiteID:  "X/USA-NY31101-X/NY31101/US/10001/12345",
		XmltvID: "WABC.us",
		Lang:    "en",
	}

	programs, err := zap2it.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Good Morning America" {
		t.Fatalf("expected title %q, got %q", "Good Morning America", p.Title)
	}
	if p.SubTitle != "Episode 1" {
		t.Fatalf("expected subtitle %q, got %q", "Episode 1", p.SubTitle)
	}
	if p.Description != "Morning news show" {
		t.Fatalf("expected description %q, got %q", "Morning news show", p.Description)
	}
	if p.Category != "News" {
		t.Fatalf("expected category %q, got %q", "News", p.Category)
	}
	if p.Season != 3 {
		t.Fatalf("expected season 3, got %d", p.Season)
	}
	if p.Episode != 12 {
		t.Fatalf("expected episode 12, got %d", p.Episode)
	}
	if p.Image != "https://zap2it.tmsimg.com/assets/p12345.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	expectedStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	p2 := programs[1]
	if p2.Title != "The View" {
		t.Fatalf("expected title %q, got %q", "The View", p2.Title)
	}
	if p2.SubTitle != "" {
		t.Fatalf("expected empty subtitle, got %q", p2.SubTitle)
	}
	if p2.Image != "" {
		t.Fatalf("expected empty image, got %q", p2.Image)
	}
	if p2.Season != 0 {
		t.Fatalf("expected season 0, got %d", p2.Season)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := zap2it.Parse([]byte("not json"), domain.Channel{})
	if err != nil {
		t.Fatalf("expected nil error for invalid JSON, got: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyResponse(t *testing.T) {
	programs, err := zap2it.Parse([]byte("{}"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
