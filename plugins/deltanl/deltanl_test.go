package deltanl_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/deltanl"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseGuide(t *testing.T) {
	data, err := os.ReadFile(testdataPath("guide.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "webtv.delta.nl-1",
		XmltvID: "NPO1.nl",
		SiteID:  "1",
		Lang:    "nl",
	}

	programs, err := deltanl.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "NOS Journaal" {
		t.Fatalf("expected title 'NOS Journaal', got %q", p.Title)
	}

	expectedStart := time.Date(2021, 11, 11, 23, 56, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2021, 11, 12, 0, 22, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image != "https://cdn.gvidi.tv/img/booxmedia/e19c/static/NOS%20Journaal5.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	if p.Channel != "NPO1.nl" {
		t.Fatalf("expected channel 'NPO1.nl', got %q", p.Channel)
	}

	if p.Source != "webtv.delta.nl" {
		t.Fatalf("expected source 'webtv.delta.nl', got %q", p.Source)
	}

	if p.Lang != "nl" {
		t.Fatalf("expected lang 'nl', got %q", p.Lang)
	}
}

func TestParseErrorResponse(t *testing.T) {
	data, err := os.ReadFile(testdataPath("error.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID: "1",
	}

	programs, err := deltanl.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := deltanl.Parse([]byte("not json"), domain.Channel{SiteID: "1"})
	if err != nil {
		t.Fatalf("expected nil error for invalid json, got: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseMissingSiteID(t *testing.T) {
	data := []byte(`{"2":[{"ID":"test","title":"Test","start":1636674960,"end":1636676520}]}`)

	programs, err := deltanl.Parse(data, domain.Channel{SiteID: "1"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyChannel(t *testing.T) {
	data := []byte(`{"1":[]}`)

	programs, err := deltanl.Parse(data, domain.Channel{SiteID: "1"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
