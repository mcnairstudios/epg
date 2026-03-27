package distrotv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/distrotv"
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
		SiteID:  "45143",
		XmltvID: "Euronews.distro.tv",
		Lang:    "en",
	}

	programs, err := distrotv.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "News" {
		t.Fatalf("expected title 'News', got %q", p.Title)
	}
	if p.Description != "The leading news stories of the moment." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	expectedStart := time.Date(2026, 2, 10, 10, 17, 5, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2026, 2, 10, 10, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if programs[1].Title != "Euronews Now" {
		t.Fatalf("expected second title 'Euronews Now', got %q", programs[1].Title)
	}
}

func TestParseEmptySlots(t *testing.T) {
	data := []byte(`{"epg":{"45143":{"title":"Euronews","slots":[]}}}`)
	programs, err := distrotv.Parse(data, domain.Channel{SiteID: "45143"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseMissingChannel(t *testing.T) {
	data := []byte(`{"epg":{}}`)
	programs, err := distrotv.Parse(data, domain.Channel{SiteID: "99999"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if programs != nil {
		t.Fatalf("expected nil programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := distrotv.Parse([]byte("not json"), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
