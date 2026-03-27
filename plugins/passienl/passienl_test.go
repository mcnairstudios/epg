package passienl_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/passienl"
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

	date := time.Date(2025, 8, 26, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "passienl-passie",
		XmltvID: "Passie.nl",
		SiteID:  "passie",
		Lang:    "nl",
	}

	programs, err := passienl.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 45 {
		t.Fatalf("expected 45 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Young Anal Charm - Scene 1" {
		t.Fatalf("expected title 'Young Anal Charm - Scene 1', got %q", p.Title)
	}

	expectedStart := time.Date(2025, 8, 25, 21, 43, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 8, 25, 22, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	last := programs[44]
	if last.Title != "Stud Doctor Slams Slutty Nurse" {
		t.Fatalf("expected last title 'Stud Doctor Slams Slutty Nurse', got %q", last.Title)
	}

	expectedLastStart := time.Date(2025, 8, 26, 21, 45, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}

	expectedLastStop := time.Date(2025, 8, 26, 22, 0, 0, 0, time.UTC)
	if !last.Stop.Equal(expectedLastStop) {
		t.Fatalf("expected last stop %v, got %v", expectedLastStop, last.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2025, 8, 26, 0, 0, 0, 0, time.UTC)
	programs, err := passienl.Parse(data, domain.Channel{}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
