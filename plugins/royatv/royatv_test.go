package royatv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/royatv"
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

	date := time.Date(2025, 8, 13, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "royatv-3",
		XmltvID: "RoyaComedy.jo",
		SiteID:  "3",
		Lang:    "ar",
	}

	programs, err := royatv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 52 {
		t.Fatalf("expected 52 programs, got %d", len(programs))
	}

	p := programs[0]

	expectedStart := time.Date(2025, 8, 12, 21, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 8, 12, 21, 36, 11, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "RoyaComedy.jo" {
		t.Fatalf("expected channel 'RoyaComedy.jo', got %q", p.Channel)
	}

	if p.Source != "roya-tv.com" {
		t.Fatalf("expected source 'roya-tv.com', got %q", p.Source)
	}

	last := programs[51]
	expectedLastStart := time.Date(2025, 8, 13, 20, 40, 19, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := royatv.Parse(data, domain.Channel{SiteID: "3"}, time.Date(2025, 8, 13, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := royatv.Parse([]byte("not json"), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
