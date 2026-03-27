package turksatkablo_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/turksatkablo"
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
		ID:      "turksatkablo.com.tr-1908",
		XmltvID: "SinemaTV.tr",
		SiteID:  "1908",
		Lang:    "tr",
	}

	date := time.Date(2025, 5, 30, 0, 0, 0, 0, time.UTC)
	programs, err := turksatkablo.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 14 {
		t.Fatalf("expected 14 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "-" {
		t.Fatalf("expected title '-', got %q", p.Title)
	}

	expectedStart := time.Date(2025, 5, 29, 21, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 5, 29, 22, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	p1 := programs[1]
	if p1.Title != "Şeytanın Evi" {
		t.Fatalf("expected title 'Şeytanın Evi', got %q", p1.Title)
	}

	if p1.Channel != "SinemaTV.tr" {
		t.Fatalf("expected channel 'SinemaTV.tr', got %q", p1.Channel)
	}

	if p1.Source != "turksatkablo.com.tr" {
		t.Fatalf("expected source 'turksatkablo.com.tr', got %q", p1.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := turksatkablo.Parse(data, domain.Channel{SiteID: "1908"}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := turksatkablo.Parse([]byte(`not json`), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
