package nhlcom_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/nhlcom"
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
		ID:      "nhl-network",
		XmltvID: "NHLNetwork.us",
		SiteID:  "nhl",
		Lang:    "en",
	}

	programs, err := nhlcom.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "On The Fly" {
		t.Fatalf("unexpected title: %q", p.Title)
	}

	if p.Category != "Sports" {
		t.Fatalf("expected category 'Sports', got %q", p.Category)
	}

	expectedStart := time.Date(2024, 11, 22, 0, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := nhlcom.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseEmptyBroadcasts(t *testing.T) {
	programs, err := nhlcom.Parse([]byte(`{"broadcasts":[]}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
