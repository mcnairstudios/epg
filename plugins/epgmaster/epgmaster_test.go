package epgmaster_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/epgmaster"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

var channel = domain.Channel{
	ID:      "epgmaster.com-ntv",
	XmltvID: "NTV.np",
	SiteID:  "ntv",
	Lang:    "ne",
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2025, 5, 18, 0, 0, 0, 0, time.UTC)
	programs, err := epgmaster.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 46 {
		t.Fatalf("expected 46 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Krishi Teleflim-Bharosa Yuwama" {
		t.Fatalf("expected title 'Krishi Teleflim-Bharosa Yuwama', got %q", p.Title)
	}

	expectedStart := time.Date(2025, 5, 18, 0, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 5, 18, 0, 15, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	p1 := programs[1]
	if p1.Title != "News in Nepali [Rec.]" {
		t.Fatalf("expected title 'News in Nepali [Rec.]', got %q", p1.Title)
	}

	expectedStart1 := time.Date(2025, 5, 18, 0, 15, 0, 0, time.UTC)
	if !p1.Start.Equal(expectedStart1) {
		t.Fatalf("expected start %v, got %v", expectedStart1, p1.Start)
	}

	if p.Channel != "NTV.np" {
		t.Fatalf("expected channel 'NTV.np', got %q", p.Channel)
	}

	if p.Source != "epgmaster.com" {
		t.Fatalf("expected source 'epgmaster.com', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := epgmaster.Parse([]byte{}, channel, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := epgmaster.Parse([]byte("not json"), channel, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
