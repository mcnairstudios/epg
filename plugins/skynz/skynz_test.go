package skynz_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/skynz"
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
		ID:      "skynz-36",
		XmltvID: "SkyMoviesFamily.nz",
		SiteID:  "36",
		Lang:    "en",
	}

	programs, err := skynz.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Sing 2" {
		t.Fatalf("expected title 'Sing 2', got %q", p.Title)
	}
	if p.Category != "Animated" {
		t.Fatalf("expected category 'Animated', got %q", p.Category)
	}

	expectedStart := time.Date(2023, 1, 20, 23, 41, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
	expectedStop := time.Date(2023, 1, 21, 1, 28, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	p5 := programs[5]
	if p5.Title != "Harry Potter and the Goblet of Fire" {
		t.Fatalf("expected title 'Harry Potter and the Goblet of Fire', got %q", p5.Title)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := skynz.Parse([]byte(`{"code":"DATE_FORMAT_ERROR"}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := skynz.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatal("expected no error")
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
