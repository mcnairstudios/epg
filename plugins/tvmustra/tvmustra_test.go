package tvmustra_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/plugins/tvmustra"
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

	date := time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)
	programs, err := tvmustra.Parse(data, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 98 {
		t.Fatalf("expected 98 programs, got %d", len(programs))
	}

	p := programs[0]
	expectedStart := time.Date(2025, 1, 17, 5, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 1, 17, 5, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Source != "tvmustra.hu" {
		t.Fatalf("expected source 'tvmustra.hu', got %q", p.Source)
	}

	last := programs[97]
	expectedLastStart := time.Date(2025, 1, 18, 4, 0, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := tvmustra.Parse(
		[]byte(""),
		time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
