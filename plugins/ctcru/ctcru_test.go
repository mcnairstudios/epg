package ctcru_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/ctcru"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

var channel = domain.Channel{
	ID:      "ctc.ru-ctc",
	XmltvID: "CTC.ru",
	SiteID:  "ctc",
	Lang:    "ru",
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := ctcru.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	expectedStart := time.Date(2025, 6, 22, 3, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 6, 22, 3, 55, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "CTC.ru" {
		t.Fatalf("expected channel 'CTC.ru', got %q", p.Channel)
	}

	if p.Source != "ctc.ru" {
		t.Fatalf("expected source 'ctc.ru', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	empty := `{"isActive":true,"content":[]}`
	programs, err := ctcru.Parse([]byte(empty), channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := ctcru.Parse([]byte("not json"), channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
