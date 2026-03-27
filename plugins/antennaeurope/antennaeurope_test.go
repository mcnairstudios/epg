package antennaeurope_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/antennaeurope"
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

	channel := domain.Channel{
		ID:      "antennaeurope.gr-#",
		XmltvID: "ANT1Europe.gr",
		SiteID:  "#",
		Lang:    "el",
	}

	date := time.Date(2025, 1, 21, 0, 0, 0, 0, time.UTC)
	programs, err := antennaeurope.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 16 {
		t.Fatalf("expected 16 programs, got %d", len(programs))
	}

	p := programs[0]
	expectedStart := time.Date(2025, 1, 21, 3, 45, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
	expectedStop := time.Date(2025, 1, 21, 7, 50, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	last := programs[15]
	expectedLastStart := time.Date(2025, 1, 22, 1, 30, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2025, 1, 21, 0, 0, 0, 0, time.UTC)
	programs, err := antennaeurope.Parse(data, domain.Channel{}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidHTML(t *testing.T) {
	programs, err := antennaeurope.Parse([]byte(""), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
