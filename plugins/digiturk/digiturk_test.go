package digiturk_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/digiturk"
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
		ID:      "digiturk-351",
		XmltvID: "Nickelodeon.tr",
		SiteID:  "351",
		Lang:    "tr",
	}

	date := time.Date(2025, 1, 12, 0, 0, 0, 0, time.UTC)

	programs, err := digiturk.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 57 {
		t.Fatalf("expected 57 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Sünger Bob Kare Pantolon" {
		t.Fatalf("expected title 'Sünger Bob Kare Pantolon', got %q", p.Title)
	}

	expectedStart := time.Date(2025, 1, 11, 21, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 1, 11, 21, 25, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	last := programs[56]
	if last.Title != "Casagrande Ailesi" {
		t.Fatalf("expected last title 'Casagrande Ailesi', got %q", last.Title)
	}

	expectedLastStart := time.Date(2025, 1, 12, 17, 40, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}

	expectedLastStop := time.Date(2025, 1, 12, 18, 0, 0, 0, time.UTC)
	if !last.Stop.Equal(expectedLastStop) {
		t.Fatalf("expected last stop %v, got %v", expectedLastStop, last.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	channel := domain.Channel{SiteID: "351"}
	date := time.Date(2025, 1, 12, 0, 0, 0, 0, time.UTC)

	programs, err := digiturk.Parse([]byte(""), channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
