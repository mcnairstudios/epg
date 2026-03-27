package cosmote_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/cosmote"
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
		ID:      "cosmote-vouli",
		XmltvID: "HellenicParliamentTV.gr",
		SiteID:  "vouli",
		Lang:    "el",
	}

	programs, err := cosmote.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]

	expected := "\u03a4\u03b9 \u039b\u03ad\u03b5\u03b9 \u03bf \u039d\u03cc\u03bc\u03bf\u03c2"
	if p.Title != expected {
		t.Fatalf("expected title %q, got %q", expected, p.Title)
	}

	if p.Category != "Special" {
		t.Fatalf("expected category %q, got %q", "Special", p.Category)
	}

	if p.Start.UTC().Format("2006-01-02T15:04:05.000Z") != "2024-12-26T23:00:00.000Z" {
		t.Fatalf("expected start 2024-12-26T23:00:00.000Z, got %v", p.Start.UTC().Format("2006-01-02T15:04:05.000Z"))
	}

	if p.Stop.UTC().Format("2006-01-02T15:04:05.000Z") != "2024-12-27T00:00:00.000Z" {
		t.Fatalf("expected stop 2024-12-27T00:00:00.000Z, got %v", p.Stop.UTC().Format("2006-01-02T15:04:05.000Z"))
	}

	if p.Image != "https://gr-ermou-prod-cache05.static.cdn.cosmotetvott.gr/ote-prod/70/280/040029714812000800_1734415727199.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{SiteID: "vouli", XmltvID: "HellenicParliamentTV.gr"}
	programs, err := cosmote.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := cosmote.Parse([]byte("not json"), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
