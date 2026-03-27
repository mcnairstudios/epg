package freeviewuk_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/freeviewuk"
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
		ID:      "freeview-4164",
		XmltvID: "BBCOneLondon.uk",
		SiteID:  "64257#4164",
		Lang:    "en",
	}

	programs, err := freeviewuk.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 25 {
		t.Fatalf("expected 25 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "The Weakest Link" {
		t.Fatalf("expected title %q, got %q", "The Weakest Link", p.Title)
	}
	if p.SubTitle != "Series 4: Episode 7" {
		t.Fatalf("expected subtitle %q, got %q", "Series 4: Episode 7", p.SubTitle)
	}
	if p.Image != "https://img.freeviewplay.tv/p0b041486e4378cbf074511098f74e78f?w=800" {
		t.Fatalf("expected image %q, got %q", "https://img.freeviewplay.tv/p0b041486e4378cbf074511098f74e78f?w=800", p.Image)
	}
	if p.Start.UTC().Format("2006-01-02T15:04:05.000Z") != "2025-01-16T00:00:00.000Z" {
		t.Fatalf("expected start 2025-01-16T00:00:00.000Z, got %v", p.Start.UTC())
	}
	if p.Stop.UTC().Format("2006-01-02T15:04:05.000Z") != "2025-01-16T00:45:00.000Z" {
		t.Fatalf("expected stop 2025-01-16T00:45:00.000Z, got %v", p.Stop.UTC())
	}

	last := programs[24]
	if last.Title != "Newscast" {
		t.Fatalf("expected last title %q, got %q", "Newscast", last.Title)
	}
	if last.Start.UTC().Format("2006-01-02T15:04:05.000Z") != "2025-01-16T23:40:00.000Z" {
		t.Fatalf("expected last start 2025-01-16T23:40:00.000Z, got %v", last.Start.UTC())
	}
	if last.Stop.UTC().Format("2006-01-02T15:04:05.000Z") != "2025-01-17T00:10:00.000Z" {
		t.Fatalf("expected last stop 2025-01-17T00:10:00.000Z, got %v", last.Stop.UTC())
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := freeviewuk.Parse([]byte(`[]`), domain.Channel{SiteID: "64257#4164"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if programs != nil && len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := freeviewuk.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatalf("expected nil error on invalid JSON, got %v", err)
	}
	if programs != nil && len(programs) != 0 {
		t.Fatalf("expected nil/empty programs, got %d", len(programs))
	}
}
