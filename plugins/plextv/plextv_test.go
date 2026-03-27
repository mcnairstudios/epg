package plextv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/plextv"
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
		SiteID:  "5eea605674085f0040ddc7a6",
		XmltvID: "Heartland.plex.tv",
		Lang:    "en",
	}

	programs, err := plextv.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Heartland" {
		t.Fatalf("expected title 'Heartland', got %q", p.Title)
	}

	if p.Channel != "Heartland.plex.tv" {
		t.Fatalf("expected channel 'Heartland.plex.tv', got %q", p.Channel)
	}

	if p.Source != "plex.tv" {
		t.Fatalf("expected source 'plex.tv', got %q", p.Source)
	}

	if p.Season == 0 {
		t.Fatal("expected non-zero season")
	}

	if p.Episode == 0 {
		t.Fatal("expected non-zero episode")
	}

	if p.Description == "" {
		t.Fatal("expected non-empty description")
	}

	hasHeartland := false
	for _, prog := range programs {
		if prog.Title == "Heartland" && prog.SubTitle != "" {
			hasHeartland = true
			break
		}
	}
	if !hasHeartland {
		t.Fatal("expected at least one Heartland program with subtitle")
	}
}

func TestParseEmpty(t *testing.T) {
	data := []byte(`{"MediaContainer":{"Metadata":[]}}`)
	programs, err := plextv.Parse(data, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalid(t *testing.T) {
	_, err := plextv.Parse([]byte("not json"), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
