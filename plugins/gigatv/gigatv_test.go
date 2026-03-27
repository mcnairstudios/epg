package gigatv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/gigatv"
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
		XmltvID: "ThainessTV.th",
		SiteID:  "222",
		Lang:    "th",
	}

	date := time.Date(2025, 1, 12, 0, 0, 0, 0, time.UTC)
	programs, err := gigatv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 32 {
		t.Fatalf("expected 32 programs, got %d", len(programs))
	}

	expectedStart := time.Date(2025, 1, 12, 0, 0, 0, 0, time.UTC)
	if !programs[0].Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, programs[0].Start)
	}

	expectedStop := time.Date(2025, 1, 12, 0, 30, 0, 0, time.UTC)
	if !programs[0].Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, programs[0].Stop)
	}

	if programs[0].Title != "THAILAND FORM ABOVE : TAK" {
		t.Fatalf("expected title 'THAILAND FORM ABOVE : TAK', got %q", programs[0].Title)
	}

	last := programs[31]
	expectedLastStart := time.Date(2025, 1, 12, 23, 30, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}

	if last.Title != "MAESA ELEPHANT CAMP" {
		t.Fatalf("expected last title 'MAESA ELEPHANT CAMP', got %q", last.Title)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := gigatv.Parse([]byte(""), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
