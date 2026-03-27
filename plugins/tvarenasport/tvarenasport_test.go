package tvarenasport_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvarenasport"
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
		XmltvID: "ArenaSport1Premium.rs",
		SiteID:  "a1p",
		Lang:    "sr",
	}

	date := time.Date(2025, 7, 30, 0, 0, 0, 0, time.UTC)
	programs, err := tvarenasport.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 13 {
		t.Fatalf("expected 13 programs, got %d", len(programs))
	}

	if programs[4].Title != "UEFA LIGA ŠAMPIONA: Liga Šampiona: Pregled sezone" {
		t.Fatalf("unexpected title at index 4: %q", programs[4].Title)
	}

	expectedStart := time.Date(2025, 7, 30, 8, 0, 0, 0, time.UTC)
	if !programs[4].Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, programs[4].Start)
	}

	expectedStop := time.Date(2025, 7, 30, 9, 0, 0, 0, time.UTC)
	if !programs[4].Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, programs[4].Stop)
	}

	if programs[4].Category != "Fudbal" {
		t.Fatalf("expected category 'Fudbal', got %q", programs[4].Category)
	}
}

func TestParseEmptyContent(t *testing.T) {
	channel := domain.Channel{SiteID: "a1p"}
	date := time.Date(2025, 7, 30, 0, 0, 0, 0, time.UTC)

	programs, err := tvarenasport.Parse([]byte(""), channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
