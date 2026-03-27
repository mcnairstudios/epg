package tvpostlu_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvpostlu"
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
		ID:      "tv.post.lu-269695d0-8076-11e9-b5ca-f345a2ed0fbe",
		XmltvID: "DasErste.de",
		SiteID:  "269695d0-8076-11e9-b5ca-f345a2ed0fbe",
		Lang:    "de",
	}

	programs, err := tvpostlu.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Tagesschau" {
		t.Fatalf("expected title 'Tagesschau', got %q", p.Title)
	}

	if p.Description != "Das Flaggschiff unter den deutschen Nachrichtensendungen ist gleichzeitig die \"dienstälteste\" noch bestehende Sendung im deutschen Fernsehen. In bis zu 20 am Tag produzierten Sendungen wird die Komplexität des Weltgeschehens verständlich erklärt und in komprimierter Form über aktuelle politische, wirtschaftliche, soziale, kulturelle, sportliche und sonstige Ereignisse berichtet." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	if p.Category != "Nachrichten" {
		t.Fatalf("expected category 'Nachrichten', got %q", p.Category)
	}

	if p.Image != "https://mp-photos-cdn.azureedge.net/container3cc71e4948ac40ab803c26e0abc2e3e5/original/e6eb49013a822f5c6eb2e7701e69a1f80aa0b947.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	expectedStart := time.Date(2023, 1, 16, 0, 5, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 1, 16, 0, 10, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "DasErste.de" {
		t.Fatalf("expected channel 'DasErste.de', got %q", p.Channel)
	}

	if p.Source != "tv.post.lu" {
		t.Fatalf("expected source 'tv.post.lu', got %q", p.Source)
	}

	if p.Lang != "de" {
		t.Fatalf("expected lang 'de', got %q", p.Lang)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := tvpostlu.Parse([]byte{}, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := tvpostlu.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseNoEPG(t *testing.T) {
	programs, err := tvpostlu.Parse([]byte(`{"result":{}}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNullResult(t *testing.T) {
	programs, err := tvpostlu.Parse([]byte(`{}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
