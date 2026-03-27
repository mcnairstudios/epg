package plutotv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/plutotv"
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
		ID:      "pluto-5ee92e72fb286e0007285fec",
		XmltvID: "Naruto.pluto.tv",
		SiteID:  "5ee92e72fb286e0007285fec",
		Lang:    "es",
	}

	programs, err := plutotv.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Naruto: El Tercer Hokage, Eternamente" {
		t.Fatalf("expected title 'Naruto: El Tercer Hokage, Eternamente', got %q", p.Title)
	}

	if p.SubTitle != "El Tercer Hokage, Eternamente" {
		t.Fatalf("expected subtitle 'El Tercer Hokage, Eternamente', got %q", p.SubTitle)
	}

	expectedStart := time.Date(2024, 12, 28, 0, 21, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2024, 12, 28, 0, 48, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Season != 2 {
		t.Fatalf("expected season 2, got %d", p.Season)
	}

	if p.Episode != 80 {
		t.Fatalf("expected episode 80, got %d", p.Episode)
	}

	if p.Category != "Anime" {
		t.Fatalf("expected category 'Anime', got %q", p.Category)
	}

	if p.Channel != "Naruto.pluto.tv" {
		t.Fatalf("expected channel 'Naruto.pluto.tv', got %q", p.Channel)
	}

	if p.Source != "pluto.tv" {
		t.Fatalf("expected source 'pluto.tv', got %q", p.Source)
	}

	if p.Image == "" {
		t.Fatal("expected image to be set")
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := plutotv.Parse([]byte(`{"timelines":[]}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := plutotv.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseNoTimelines(t *testing.T) {
	programs, err := plutotv.Parse([]byte(`{}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
