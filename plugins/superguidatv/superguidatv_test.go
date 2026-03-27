package superguidatv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/superguidatv"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "superguidatv-virgin-radio/461",
		XmltvID: "VirginRadioTV.it",
		SiteID:  "virgin-radio/461",
		Lang:    "it",
	}

	programs, err := superguidatv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "All Nite Rock" {
		t.Fatalf("expected title 'All Nite Rock', got %q", p.Title)
	}
	if p.Category != "Musica" {
		t.Fatalf("expected category 'Musica', got %q", p.Category)
	}

	expectedStart := time.Date(2023, 1, 11, 1, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 1, 11, 5, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "VirginRadioTV.it" {
		t.Fatalf("expected channel 'VirginRadioTV.it', got %q", p.Channel)
	}

	if p.Source != "superguidatv.it" {
		t.Fatalf("expected source 'superguidatv.it', got %q", p.Source)
	}

	if len(programs) < 14 {
		t.Fatalf("expected at least 14 programs, got %d", len(programs))
	}

	last := programs[13]
	if last.Title != "Free Rock" {
		t.Fatalf("expected title 'Free Rock', got %q", last.Title)
	}

	expectedLastStart := time.Date(2023, 1, 12, 5, 0, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}

	expectedLastStop := time.Date(2023, 1, 12, 5, 30, 0, 0, time.UTC)
	if !last.Stop.Equal(expectedLastStop) {
		t.Fatalf("expected last stop %v, got %v", expectedLastStop, last.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{SiteID: "virgin-radio/461", Lang: "it"}

	programs, err := superguidatv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyData(t *testing.T) {
	date := time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{SiteID: "virgin-radio/461", Lang: "it"}

	programs, err := superguidatv.Parse([]byte(""), channel, date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
