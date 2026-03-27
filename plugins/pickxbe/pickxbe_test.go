package pickxbe_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/pickxbe"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("data.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "pickxbe-UID0118",
		XmltvID: "Vivacite.be",
		SiteID:  "UID0118",
		Lang:    "fr",
	}

	programs, err := pickxbe.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Le 22h30" {
		t.Fatalf("expected title 'Le 22h30', got %q", p.Title)
	}
	if p.Start.Format("2006-01-02T15:04:05.000Z") != "2023-12-12T23:55:00.000Z" {
		t.Fatalf("unexpected start: %v", p.Start)
	}
	if p.Stop.Format("2006-01-02T15:04:05.000Z") != "2023-12-13T00:15:00.000Z" {
		t.Fatalf("unexpected stop: %v", p.Stop)
	}
	if p.Description != "Le journal de vivre ici." {
		t.Fatalf("unexpected description: %q", p.Description)
	}
	if p.Category != "Info" {
		t.Fatalf("expected category 'Info', got %q", p.Category)
	}
	if p.Image != "https://experience-cache.proximustv.be/posterserver/poster/EPG/250_250_4B990CC58066A7B2A660AFA0BDDE5C41.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}
}

func TestParseEmptyContent(t *testing.T) {
	channel := domain.Channel{Lang: "fr"}
	programs, err := pickxbe.Parse([]byte(""), channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	channel := domain.Channel{Lang: "fr"}
	programs, err := pickxbe.Parse([]byte(`not json`), channel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
