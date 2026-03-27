package tvtvus_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvtvus"
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
		ID:      "tvtvus-20373",
		XmltvID: "SNY.us",
		SiteID:  "20373",
		Lang:    "en",
	}

	programs, err := tvtvus.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 33 {
		t.Fatalf("expected 33 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "NY Sports Nation Nightly" {
		t.Fatalf("expected title 'NY Sports Nation Nightly', got %q", p.Title)
	}
	expectedStart := time.Date(2025, 1, 30, 0, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
	expectedStop := time.Date(2025, 1, 30, 0, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	p1 := programs[1]
	if p1.Title != "The Big Bang Theory" {
		t.Fatalf("expected title 'The Big Bang Theory', got %q", p1.Title)
	}
	if p1.SubTitle != "The Bow Tie Asymmetry" {
		t.Fatalf("expected subtitle 'The Bow Tie Asymmetry', got %q", p1.SubTitle)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := tvtvus.Parse([]byte(`[]`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := tvtvus.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatal("expected no error")
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
