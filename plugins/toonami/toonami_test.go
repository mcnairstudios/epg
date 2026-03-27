package toonami_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/toonami"
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
		ID:      "toonami-est",
		XmltvID: "ToonamiAftermath.us",
		SiteID:  "EST",
		Lang:    "en",
	}

	programs, err := toonami.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "X-Men" {
		t.Fatalf("unexpected title: %q", p.Title)
	}

	if p.SubTitle != "Reunion (Part 1)" {
		t.Fatalf("unexpected subtitle: %q", p.SubTitle)
	}

	expectedStart := time.Date(2022, 11, 29, 17, 0, 30, 231000000, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	if p.Image == "" {
		t.Fatal("expected image to be set")
	}
}

func TestParseEmptyPlaylist(t *testing.T) {
	programs, err := toonami.Parse([]byte(`{"playlist":{"blocks":[]}}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := toonami.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
