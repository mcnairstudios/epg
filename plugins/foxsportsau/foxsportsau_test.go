package foxsportsau_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/foxsportsau"
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
		ID:      "foxsports-au-2",
		XmltvID: "FoxLeague.au",
		SiteID:  "2",
		Lang:    "en",
	}

	programs, err := foxsportsau.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "NRL" {
		t.Fatalf("expected title 'NRL', got %q", p.Title)
	}

	if p.SubTitle != "Eels v Titans" {
		t.Fatalf("expected subtitle 'Eels v Titans', got %q", p.SubTitle)
	}

	if p.Description != "The Eels and Titans have plenty of motivation this season after heartbreaking Finals losses in 2021. Parramatta has won their past five against Gold Coast." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	if p.Category != "Rugby League" {
		t.Fatalf("expected category 'Rugby League', got %q", p.Category)
	}

	expectedStart := time.Date(2022, 12, 13, 13, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 12, 13, 14, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "FoxLeague.au" {
		t.Fatalf("expected channel 'FoxLeague.au', got %q", p.Channel)
	}

	if p.Source != "foxsports.com.au" {
		t.Fatalf("expected source 'foxsports.com.au', got %q", p.Source)
	}

	if p.Lang != "en" {
		t.Fatalf("expected lang 'en', got %q", p.Lang)
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "2",
		XmltvID: "FoxLeague.au",
	}

	programs, err := foxsportsau.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := foxsportsau.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseEmptyResponse(t *testing.T) {
	programs, err := foxsportsau.Parse([]byte(`{}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseFiltersByChannelID(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "999",
		XmltvID: "Other.au",
	}

	programs, err := foxsportsau.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 0 {
		t.Fatalf("expected 0 programs for non-matching channel, got %d", len(programs))
	}
}
