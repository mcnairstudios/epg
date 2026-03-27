package mujtvprogram_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/mujtvprogram"
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
		ID:      "mujtvprogram-1",
		XmltvID: "CT1.cz",
		SiteID:  "1",
		Lang:    "cs",
	}

	programs, err := mujtvprogram.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) < 4 {
		t.Fatalf("expected at least 4 programs, got %d", len(programs))
	}

	p := programs[3]

	if p.Title != "\u010cepice" {
		t.Fatalf("expected title '\u010cepice', got %q", p.Title)
	}

	if p.Category != "film" {
		t.Fatalf("expected category 'film', got %q", p.Category)
	}

	expectedStart := time.Date(2022, 12, 23, 8, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 12, 23, 8, 20, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := mujtvprogram.Parse(data, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidXML(t *testing.T) {
	programs, err := mujtvprogram.Parse([]byte(`not xml`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse should not error, got: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
