package reportv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/reportv"
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
		ID:      "reportv.com.ar-914",
		XmltvID: "VePlusVenezuela.ve",
		SiteID:  "914",
		Lang:    "es",
	}

	date := time.Date(2022, 10, 3, 0, 0, 0, 0, time.UTC)
	programs, err := reportv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "\u00bfQui\u00e9n tiene la raz\u00f3n?" {
		t.Fatalf("expected title '¿Quién tiene la razón?', got %q", p.Title)
	}

	if p.Category != "Talk Show" {
		t.Fatalf("expected category 'Talk Show', got %q", p.Category)
	}

	expectedStart := time.Date(2022, 10, 3, 4, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 10, 3, 5, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{SiteID: "914"}
	date := time.Date(2022, 10, 3, 0, 0, 0, 0, time.UTC)

	programs, err := reportv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse should not fail: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNoData(t *testing.T) {
	programs, err := reportv.Parse([]byte(""), domain.Channel{}, time.Time{})
	if err != nil {
		t.Fatalf("parse should not fail: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
