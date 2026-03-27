package teleboy_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/teleboy"
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
		ID:      "teleboy-303",
		XmltvID: "SRF1.ch",
		SiteID:  "303",
		Lang:    "de",
	}

	programs, err := teleboy.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 35 {
		t.Fatalf("expected 35 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Der Staatsanwalt" {
		t.Fatalf("expected title 'Der Staatsanwalt', got %q", p.Title)
	}
	if p.SubTitle != "Tod eines Helden" {
		t.Fatalf("expected subtitle 'Tod eines Helden', got %q", p.SubTitle)
	}
	if p.Season != 16 {
		t.Fatalf("expected season 16, got %d", p.Season)
	}
	if p.Episode != 6 {
		t.Fatalf("expected episode 6, got %d", p.Episode)
	}
	if p.Start.Format("2006-01-02T15:04:05.000Z") != "2025-01-25T22:45:00.000Z" {
		t.Fatalf("unexpected start: %v", p.Start)
	}
	if p.Stop.Format("2006-01-02T15:04:05.000Z") != "2025-01-25T23:50:00.000Z" {
		t.Fatalf("unexpected stop: %v", p.Stop)
	}
	if p.Image != "https://media.teleboy.ch/media/teleboyteaser6/bd01aed53c7a37399ae034c2a1a2cc8aa31943f2.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{SiteID: "303", Lang: "de"}
	programs, err := teleboy.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	channel := domain.Channel{SiteID: "303", Lang: "de"}
	programs, err := teleboy.Parse([]byte(`not json`), channel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
