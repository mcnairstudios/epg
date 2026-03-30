package tvkaista_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvkaista"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent1(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_1.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	programs, err := tvkaista.Parse(data, domain.Channel{XmltvID: "Test", Lang: "xx", SiteID: "test"}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 45 {
		t.Fatalf("expected 45 programs, got %d", len(programs))
	}

	p := programs[0]
	expectedStart := time.Date(2025, 2, 28, 21, 20, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 2, 28, 22, 4, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Title != "Alice & Jack" {
		t.Fatalf("expected title 'Alice & Jack', got %q", p.Title)
	}

	if p.Season != 1 {
		t.Fatalf("expected season 1, got %d", p.Season)
	}

	if p.Episode != 2 {
		t.Fatalf("expected episode 2, got %d", p.Episode)
	}

	if p.Source != "tvkaista.org" {
		t.Fatalf("expected source 'tvkaista.org', got %q", p.Source)
	}
}

func TestParseContent2(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_2.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC)
	programs, err := tvkaista.Parse(data, domain.Channel{XmltvID: "Test", Lang: "xx", SiteID: "test"}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 39 {
		t.Fatalf("expected 39 programs, got %d", len(programs))
	}

	p := programs[0]
	expectedStart := time.Date(2025, 3, 2, 21, 5, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	programs, err := tvkaista.Parse(data, domain.Channel{XmltvID: "Test", Lang: "xx", SiteID: "test"}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
