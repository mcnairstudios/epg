package freetv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/freetv"
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
		ID:      "freetv-3370462",
		XmltvID: "Kan11.il",
		SiteID:  "3370462",
		Lang:    "he",
	}

	programs, err := freetv.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Start.UTC().Format("2006-01-02T15:04:05.000Z") != "2025-03-27T21:26:00.000Z" {
		t.Fatalf("expected start 2025-03-27T21:26:00.000Z, got %v", p.Start.UTC().Format("2006-01-02T15:04:05.000Z"))
	}

	if p.Stop.UTC().Format("2006-01-02T15:04:05.000Z") != "2025-03-27T22:17:00.000Z" {
		t.Fatalf("expected stop 2025-03-27T22:17:00.000Z, got %v", p.Stop.UTC().Format("2006-01-02T15:04:05.000Z"))
	}

	if p.Image == "" {
		t.Fatal("expected image to be set")
	}

	p2 := programs[1]
	if p2.Start.UTC().Format("2006-01-02T15:04:05.000Z") != "2025-03-27T22:17:00.000Z" {
		t.Fatalf("expected second start 2025-03-27T22:17:00.000Z, got %v", p2.Start.UTC().Format("2006-01-02T15:04:05.000Z"))
	}
	if p2.Stop.UTC().Format("2006-01-02T15:04:05.000Z") != "2025-03-27T22:43:00.000Z" {
		t.Fatalf("expected second stop 2025-03-27T22:43:00.000Z, got %v", p2.Stop.UTC().Format("2006-01-02T15:04:05.000Z"))
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := freetv.Parse([]byte(``), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if programs != nil && len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := freetv.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if programs != nil && len(programs) != 0 {
		t.Fatalf("expected nil/empty programs, got %d", len(programs))
	}
}
