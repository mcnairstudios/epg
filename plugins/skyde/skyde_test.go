package skyde_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/skyde"
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
		ID:      "skyde-522",
		XmltvID: "WarnerTVComedyHD.de",
		SiteID:  "522",
		Lang:    "de",
	}

	programs, err := skyde.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "King of Queens" {
		t.Fatalf("expected title 'King of Queens', got %q", p.Title)
	}

	if p.Description != "Der Experte" {
		t.Fatalf("expected description 'Der Experte', got %q", p.Description)
	}

	if p.Category != "Comedyserie" {
		t.Fatalf("expected category 'Comedyserie', got %q", p.Category)
	}

	expectedStart := time.Date(2022, 2, 26, 23, 5, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 2, 26, 23, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Season != 4 {
		t.Fatalf("expected season 4, got %d", p.Season)
	}

	if p.Episode != 11 {
		t.Fatalf("expected episode 11, got %d", p.Episode)
	}

	if p.Image != "http://sky.de/static/img/program_guide/1522936_s.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := skyde.Parse([]byte(`[]`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := skyde.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
