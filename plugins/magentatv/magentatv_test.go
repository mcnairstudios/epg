package magentatv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/magentatv"
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
		ID:      "magentatv-255",
		XmltvID: "13thStreet.de",
		SiteID:  "255",
		Lang:    "de",
	}

	programs, err := magentatv.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Twenty Foot Plus" {
		t.Fatalf("expected title 'Twenty Foot Plus', got %q", p.Title)
	}

	if p.Description != "Die besten Big-Wave-Surfer werden bei ihrer Suche nach der nächsten großen Welle begleitet." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	if p.Image != "http://ngiss.t-online.de/cm1s/media/gracenote/2/4/p24832950_e_h9_aa_2023-06-22T10_12_01.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	if p.Category != "Sport" {
		t.Fatalf("expected category 'Sport', got %q", p.Category)
	}

	expectedStart := time.Date(2023, 10, 23, 23, 58, 55, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 10, 24, 0, 11, 5, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "13thStreet.de" {
		t.Fatalf("expected channel '13thStreet.de', got %q", p.Channel)
	}

	if p.Source != "web.magentatv.de" {
		t.Fatalf("expected source 'web.magentatv.de', got %q", p.Source)
	}

	p2 := programs[1]

	if p2.Title != "The Big Bang Theory" {
		t.Fatalf("expected title 'The Big Bang Theory', got %q", p2.Title)
	}

	if p2.SubTitle != "Tritte unter dem Tisch" {
		t.Fatalf("expected subtitle 'Tritte unter dem Tisch', got %q", p2.SubTitle)
	}

	if p2.Season != 7 {
		t.Fatalf("expected season 7, got %d", p2.Season)
	}

	if p2.Episode != 5 {
		t.Fatalf("expected episode 5, got %d", p2.Episode)
	}

	if p2.Category != "Sitcom" {
		t.Fatalf("expected category 'Sitcom', got %q", p2.Category)
	}

	expectedStart2 := time.Date(2024, 11, 5, 15, 37, 3, 0, time.UTC)
	if !p2.Start.Equal(expectedStart2) {
		t.Fatalf("expected start %v, got %v", expectedStart2, p2.Start)
	}

	expectedStop2 := time.Date(2024, 11, 5, 16, 3, 48, 0, time.UTC)
	if !p2.Stop.Equal(expectedStop2) {
		t.Fatalf("expected stop %v, got %v", expectedStop2, p2.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := magentatv.Parse([]byte(`{"counttotal":"0"}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := magentatv.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseNoPlaybillList(t *testing.T) {
	programs, err := magentatv.Parse([]byte(`{}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
