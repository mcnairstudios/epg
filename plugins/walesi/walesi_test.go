package walesi_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/walesi"
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

	date := time.Date(2021, 11, 21, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "walesi-fbc-2",
		XmltvID: "FBCTV.fj",
		SiteID:  "fbc-2",
		Lang:    "fj",
	}

	programs, err := walesi.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Aljazeera" {
		t.Fatalf("expected title 'Aljazeera', got %q", p.Title)
	}

	expectedStart := time.Date(2021, 11, 20, 12, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2021, 11, 20, 18, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	p1 := programs[1]
	if p1.Title != "Move Fiji" {
		t.Fatalf("expected title 'Move Fiji', got %q", p1.Title)
	}

	expectedStart1 := time.Date(2021, 11, 20, 18, 0, 0, 0, time.UTC)
	if !p1.Start.Equal(expectedStart1) {
		t.Fatalf("expected start %v, got %v", expectedStart1, p1.Start)
	}

	expectedStop1 := time.Date(2021, 11, 20, 18, 30, 0, 0, time.UTC)
	if !p1.Stop.Equal(expectedStop1) {
		t.Fatalf("expected stop %v, got %v", expectedStop1, p1.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2021, 11, 21, 0, 0, 0, 0, time.UTC)
	programs, err := walesi.Parse(data, domain.Channel{}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	date := time.Date(2021, 11, 21, 0, 0, 0, 0, time.UTC)
	_, err := walesi.Parse([]byte(`not json`), domain.Channel{}, date)
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
