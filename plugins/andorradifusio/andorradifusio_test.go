package andorradifusio_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/andorradifusio"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

var channel = domain.Channel{
	ID:      "andorradifusio.ad-atv",
	XmltvID: "AndorraTV.ad",
	SiteID:  "atv",
	Lang:    "ca",
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2023, 6, 7, 0, 0, 0, 0, time.UTC)
	programs, err := andorradifusio.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Club Piolet" {
		t.Fatalf("expected title 'Club Piolet', got %q", p.Title)
	}

	expectedStart := time.Date(2023, 6, 7, 5, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 6, 7, 6, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if len(programs) < 21 {
		t.Fatalf("expected at least 21 programs, got %d", len(programs))
	}

	p20 := programs[20]
	if p20.Title != "\u00c0rea Andorra Difusi\u00f3" {
		t.Fatalf("expected title 'Area Andorra Difusio', got %q", p20.Title)
	}

	expectedStart20 := time.Date(2023, 6, 7, 23, 0, 0, 0, time.UTC)
	if !p20.Start.Equal(expectedStart20) {
		t.Fatalf("expected start %v, got %v", expectedStart20, p20.Start)
	}

	expectedStop20 := time.Date(2023, 6, 8, 0, 0, 0, 0, time.UTC)
	if !p20.Stop.Equal(expectedStop20) {
		t.Fatalf("expected stop %v, got %v", expectedStop20, p20.Stop)
	}

	if p.Channel != "AndorraTV.ad" {
		t.Fatalf("expected channel 'AndorraTV.ad', got %q", p.Channel)
	}

	if p.Source != "andorradifusio.ad" {
		t.Fatalf("expected source 'andorradifusio.ad', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := andorradifusio.Parse([]byte("<!DOCTYPE html><html><head></head><body></body></html>"), channel, time.Date(2023, 6, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
