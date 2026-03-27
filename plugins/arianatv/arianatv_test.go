package arianatv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/arianatv"
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
		XmltvID: "ATNNational.af",
		SiteID:  "#",
		Lang:    "en",
	}

	date := time.Date(2021, 11, 27, 0, 0, 0, 0, time.UTC)
	programs, err := arianatv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 4 {
		t.Fatalf("expected 4 programs, got %d", len(programs))
	}

	if programs[0].Title != "City Report" {
		t.Fatalf("expected title 'City Report', got %q", programs[0].Title)
	}

	expectedStart := time.Date(2021, 11, 27, 2, 30, 0, 0, time.UTC)
	if !programs[0].Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, programs[0].Start)
	}

	expectedStop := time.Date(2021, 11, 27, 3, 0, 0, 0, time.UTC)
	if !programs[0].Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, programs[0].Stop)
	}

	if programs[1].Title != "ICC T20 Highlights" {
		t.Fatalf("expected title 'ICC T20 Highlights', got %q", programs[1].Title)
	}

	expectedStart2 := time.Date(2021, 11, 27, 3, 0, 0, 0, time.UTC)
	if !programs[1].Start.Equal(expectedStart2) {
		t.Fatalf("expected start %v, got %v", expectedStart2, programs[1].Start)
	}

	expectedStop2 := time.Date(2021, 11, 27, 10, 30, 0, 0, time.UTC)
	if !programs[1].Stop.Equal(expectedStop2) {
		t.Fatalf("expected stop %v, got %v", expectedStop2, programs[1].Stop)
	}

	expectedStart3 := time.Date(2021, 11, 28, 2, 0, 0, 0, time.UTC)
	if !programs[3].Start.Equal(expectedStart3) {
		t.Fatalf("expected start %v, got %v", expectedStart3, programs[3].Start)
	}

	if programs[3].Title != "Quran and Hadis" {
		t.Fatalf("expected title 'Quran and Hadis', got %q", programs[3].Title)
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{SiteID: "#"}
	date := time.Date(2021, 11, 27, 0, 0, 0, 0, time.UTC)

	programs, err := arianatv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := arianatv.Parse([]byte(""), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
