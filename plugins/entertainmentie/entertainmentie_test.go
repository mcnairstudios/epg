package entertainmentie_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/entertainmentie"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

var channel = domain.Channel{
	ID:      "entertainment.ie-rte2",
	XmltvID: "RTE2.ie",
	SiteID:  "rte2",
	Lang:    "en",
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2023, 6, 29, 0, 0, 0, 0, time.UTC)
	programs, err := entertainmentie.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 51 {
		t.Fatalf("expected 51 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "EuroNews" {
		t.Fatalf("expected title 'EuroNews', got %q", p.Title)
	}

	expectedStart := time.Date(2023, 6, 29, 6, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 6, 29, 8, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Description != "European and international headlines live via satellite" {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	if p.Category != "Factual" {
		t.Fatalf("expected category 'Factual', got %q", p.Category)
	}

	p50 := programs[50]
	if p50.Title != "EuroNews" {
		t.Fatalf("expected title 'EuroNews', got %q", p50.Title)
	}

	expectedStart50 := time.Date(2023, 6, 30, 2, 25, 0, 0, time.UTC)
	if !p50.Start.Equal(expectedStart50) {
		t.Fatalf("expected start %v, got %v", expectedStart50, p50.Start)
	}

	expectedStop50 := time.Date(2023, 6, 30, 6, 0, 0, 0, time.UTC)
	if !p50.Stop.Equal(expectedStop50) {
		t.Fatalf("expected stop %v, got %v", expectedStop50, p50.Stop)
	}

	if p.Channel != "RTE2.ie" {
		t.Fatalf("expected channel 'RTE2.ie', got %q", p.Channel)
	}

	if p.Source != "entertainment.ie" {
		t.Fatalf("expected source 'entertainment.ie', got %q", p.Source)
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no-content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2023, 6, 29, 0, 0, 0, 0, time.UTC)
	programs, err := entertainmentie.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
