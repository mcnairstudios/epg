package tvgidsnl_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvgidsnl"
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

	date := time.Date(2025, 1, 19, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "tvgidsnl-npo1",
		XmltvID: "NPO1.nl",
		SiteID:  "npo1",
		Lang:    "nl",
	}

	programs, err := tvgidsnl.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Op1" {
		t.Fatalf("expected title 'Op1', got %q", p.Title)
	}

	expectedStart := time.Date(2025, 1, 18, 22, 57, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 1, 18, 23, 58, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image != "https://tvgidsassets.nl/v301/upload/o/carrousel/op1-451542641.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	if p.Channel != "NPO1.nl" {
		t.Fatalf("expected channel 'NPO1.nl', got %q", p.Channel)
	}

	if p.Source != "tvgids.nl" {
		t.Fatalf("expected source 'tvgids.nl', got %q", p.Source)
	}

	if len(programs) < 62 {
		t.Fatalf("expected at least 62 programs, got %d", len(programs))
	}

	last := programs[61]
	if last.Title != "NOS Journaal" {
		t.Fatalf("expected last title 'NOS Journaal', got %q", last.Title)
	}

	expectedLastStart := time.Date(2025, 1, 20, 1, 18, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := tvgidsnl.Parse([]byte(""), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
