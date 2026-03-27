package vidio_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/vidio"
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
		ID:      "vidio-204",
		XmltvID: "SCTV.id",
		SiteID:  "204",
		Lang:    "id",
	}

	programs, err := vidio.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 21 {
		t.Fatalf("expected 21 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Ftv PrimeTime : Cinta Dodol Inilah Yang Membuatku Lengket Padamu" {
		t.Fatalf("expected title 'Ftv PrimeTime : Cinta Dodol Inilah Yang Membuatku Lengket Padamu', got %q", p.Title)
	}

	if p.Description != "Film televisi yang mengangkat kisah romantisme kehidupan dengan konflik yang menarik. tayang setiap hari" {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	expectedStart := time.Date(2025, 6, 30, 15, 57, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 6, 30, 17, 29, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "SCTV.id" {
		t.Fatalf("expected channel 'SCTV.id', got %q", p.Channel)
	}

	if p.Source != "vidio.com" {
		t.Fatalf("expected source 'vidio.com', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := vidio.Parse(data, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := vidio.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
