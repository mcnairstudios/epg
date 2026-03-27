package webtvsk_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/webtvsk"
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
		ID:      "webtv.sk-golf_channel",
		XmltvID: "GolfChannel.sk",
		SiteID:  "golf_channel",
		Lang:    "sk",
	}

	programs, err := webtvsk.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 12 {
		t.Fatalf("expected 12 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Qatar Masters" {
		t.Fatalf("expected title 'Qatar Masters', got %q", p.Title)
	}

	if p.Category != "\u0161port" {
		t.Fatalf("expected category 'šport', got %q", p.Category)
	}

	expectedStart := time.Date(2026, 2, 16, 23, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2026, 2, 17, 4, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	last := programs[11]
	if last.Title != "The Golf Fix" {
		t.Fatalf("expected last title 'The Golf Fix', got %q", last.Title)
	}

	lastStart := time.Date(2026, 2, 17, 22, 30, 0, 0, time.UTC)
	if !last.Start.Equal(lastStart) {
		t.Fatalf("expected last start %v, got %v", lastStart, last.Start)
	}

	lastStop := time.Date(2026, 2, 17, 23, 0, 0, 0, time.UTC)
	if !last.Stop.Equal(lastStop) {
		t.Fatalf("expected last stop %v, got %v", lastStop, last.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := webtvsk.Parse([]byte(`{}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := webtvsk.Parse([]byte(``), domain.Channel{})
	if err != nil {
		t.Fatalf("parse should not fail on empty: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
