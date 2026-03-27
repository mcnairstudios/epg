package astro_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/astro"
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
		ID:      "astro-425",
		XmltvID: "TVBClassic.hk",
		SiteID:  "425",
		Lang:    "ms",
	}

	date := time.Date(2022, 10, 31, 0, 0, 0, 0, time.UTC)
	programs, err := astro.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 31 {
		t.Fatalf("expected 31 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Triumph in the Skies Ep06" {
		t.Fatalf("expected title %q, got %q", "Triumph in the Skies Ep06", p.Title)
	}

	if p.Episode != 6 {
		t.Fatalf("expected episode 6, got %d", p.Episode)
	}

	expectedStart := time.Date(2022, 10, 30, 16, 10, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 10, 30, 17, 2, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2022, 10, 31, 0, 0, 0, 0, time.UTC)
	programs, err := astro.Parse(data, domain.Channel{SiteID: "425"}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if programs != nil && len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	date := time.Date(2022, 10, 31, 0, 0, 0, 0, time.UTC)
	programs, err := astro.Parse([]byte("not json"), domain.Channel{}, date)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if programs != nil && len(programs) != 0 {
		t.Fatalf("expected nil/empty programs, got %d", len(programs))
	}
}
