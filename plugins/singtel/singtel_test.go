package singtel_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/singtel"
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
		ID:      "singtel-5418",
		XmltvID: "ParamountNetworkSingapore.sg",
		SiteID:  "5418",
		Lang:    "en",
	}

	programs, err := singtel.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 23 {
		t.Fatalf("expected 23 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Hip Hop Family Christmas Wedding" {
		t.Fatalf("expected title 'Hip Hop Family Christmas Wedding', got %q", p.Title)
	}

	if p.Description != "Hip Hop's most famous family is back, and this time Christmas wedding bells are ringing! Jessica and Jayson are getting ready to say their \"I do's\"." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	if p.Category != "Specials" {
		t.Fatalf("expected category 'Specials', got %q", p.Category)
	}

	singapore := time.FixedZone("Asia/Singapore", 8*60*60)
	expectedStart := time.Date(2023, 1, 29, 0, 0, 0, 0, singapore)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 1, 29, 1, 30, 0, 0, singapore)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "ParamountNetworkSingapore.sg" {
		t.Fatalf("expected channel 'ParamountNetworkSingapore.sg', got %q", p.Channel)
	}

	if p.Source != "singtel.com" {
		t.Fatalf("expected source 'singtel.com', got %q", p.Source)
	}

	if p.Lang != "en" {
		t.Fatalf("expected lang 'en', got %q", p.Lang)
	}

	p10 := programs[10]

	if p10.Title != "The Daily Show" {
		t.Fatalf("expected title 'The Daily Show', got %q", p10.Title)
	}

	if p10.Category != "English Entertainment" {
		t.Fatalf("expected category 'English Entertainment', got %q", p10.Category)
	}

	expectedStart10 := time.Date(2023, 1, 29, 9, 0, 0, 0, singapore)
	if !p10.Start.Equal(expectedStart10) {
		t.Fatalf("expected start %v, got %v", expectedStart10, p10.Start)
	}

	expectedStop10 := time.Date(2023, 1, 29, 9, 30, 0, 0, singapore)
	if !p10.Stop.Equal(expectedStop10) {
		t.Fatalf("expected stop %v, got %v", expectedStop10, p10.Stop)
	}
}

func TestParseMissingSiteID(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{SiteID: "9999"}
	programs, err := singtel.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := singtel.Parse([]byte(`{}`), domain.Channel{SiteID: "5418"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := singtel.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
