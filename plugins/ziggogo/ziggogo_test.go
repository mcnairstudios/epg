package ziggogo_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/ziggogo"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseProgram01(t *testing.T) {
	data, err := os.ReadFile(testdataPath("program01.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "ziggogo.tv-NL_000001_019401",
		XmltvID: "NPO1.nl",
		SiteID:  "NL_000001_019401",
		Lang:    "nl",
	}

	p, err := ziggogo.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if p.Title != "EenVandaag" {
		t.Fatalf("expected title 'EenVandaag', got %q", p.Title)
	}

	if p.Description != "Op pad met HTS-rebellen in Syrië. Nieuwe aanpak tegen te veel zitten. VS heeft Tiktok-ban bijna rond. Wat is de rol van Nederland in de onderhandeling rondom Oekraïne?" {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	if p.Category != "Nieuws, Actualiteit" {
		t.Fatalf("expected category 'Nieuws, Actualiteit', got %q", p.Category)
	}

	if p.Season != 11 {
		t.Fatalf("expected season 11, got %d", p.Season)
	}

	if p.Episode != 300 {
		t.Fatalf("expected episode 300, got %d", p.Episode)
	}

	expectedStart := time.Date(2024, 12, 17, 0, 10, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2024, 12, 17, 0, 35, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "NPO1.nl" {
		t.Fatalf("expected channel 'NPO1.nl', got %q", p.Channel)
	}

	if p.Source != "ziggogo.tv" {
		t.Fatalf("expected source 'ziggogo.tv', got %q", p.Source)
	}
}

func TestParseProgram03(t *testing.T) {
	data, err := os.ReadFile(testdataPath("program03.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "ziggogo.tv-NL_000001_019401",
		XmltvID: "NPO1.nl",
		SiteID:  "NL_000001_019401",
		Lang:    "nl",
	}

	p, err := ziggogo.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if p.Title != "Bar Laat" {
		t.Fatalf("expected title 'Bar Laat', got %q", p.Title)
	}

	if p.Category != "Talkshow" {
		t.Fatalf("expected category 'Talkshow', got %q", p.Category)
	}

	if p.Season != 1 {
		t.Fatalf("expected season 1, got %d", p.Season)
	}

	if p.Episode != 65 {
		t.Fatalf("expected episode 65, got %d", p.Episode)
	}

	expectedStart := time.Date(2024, 12, 17, 14, 55, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2024, 12, 17, 15, 58, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	channel := domain.Channel{
		XmltvID: "NPO1.nl",
		Lang:    "nl",
	}

	_, err := ziggogo.Parse([]byte(`not json`), channel)
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseEmptyObject(t *testing.T) {
	channel := domain.Channel{
		XmltvID: "NPO1.nl",
		Lang:    "nl",
	}

	p, err := ziggogo.Parse([]byte(`{}`), channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if p.Title != "" {
		t.Fatalf("expected empty title, got %q", p.Title)
	}
}
