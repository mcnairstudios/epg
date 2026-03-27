package cosmote_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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
		SiteID:  "vouli",
		XmltvID: "HellenicParliamentTV.gr",
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

	if p.Title != "Τι Λέει ο Νόμος" {
		t.Fatalf("expected title 'Τι Λέει ο Νόμος', got %q", p.Title)
	}

	if p.Category != "Special" {
		t.Fatalf("expected category 'Special', got %q", p.Category)
	}

	if p.Image != "https://gr-ermou-prod-cache05.static.cdn.cosmotetvott.gr/ote-prod/70/280/040029714812000800_1734415727199.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	expectedStart := time.Date(2024, 12, 26, 23, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2024, 12, 27, 0, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "vouli",
		XmltvID: "HellenicParliamentTV.gr",
		Lang:    "el",
	}

	programs, err := cosmote.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := cosmote.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
