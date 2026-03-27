package airtelxstream_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/airtelxstream"
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
		ID:      "airtelxstream.in-LIVETV_LIVETVCHANNEL_ZEE_CINEMA",
		XmltvID: "ZeeCinema.in",
		SiteID:  "LIVETV_LIVETVCHANNEL_ZEE_CINEMA",
		Lang:    "en",
	}

	programs, err := airtelxstream.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Bengal Tiger" {
		t.Fatalf("expected title 'Bengal Tiger', got %q", p.Title)
	}

	if p.Description == "" {
		t.Fatal("expected description to be set")
	}

	expectedStart := time.UnixMilli(1766868120000)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.UnixMilli(1766875380000)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image == "" {
		t.Fatal("expected image to be set")
	}

	if p.Channel != "ZeeCinema.in" {
		t.Fatalf("expected channel 'ZeeCinema.in', got %q", p.Channel)
	}

	if p.Source != "airtelxstream.in" {
		t.Fatalf("expected source 'airtelxstream.in', got %q", p.Source)
	}
}

func TestParseEmptyGuide(t *testing.T) {
	programs, err := airtelxstream.Parse([]byte(`{"programGuide":{}}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := airtelxstream.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseNoProgramGuide(t *testing.T) {
	programs, err := airtelxstream.Parse([]byte(`{}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseMultiplePrograms(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "LIVETV_LIVETVCHANNEL_ZEE_CINEMA",
		XmltvID: "ZeeCinema.in",
		Lang:    "en",
	}

	programs, err := airtelxstream.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 10 {
		t.Fatalf("expected 10 programs, got %d", len(programs))
	}

	last := programs[len(programs)-1]
	if last.Title != "Sooryavanshi" {
		t.Fatalf("expected last title 'Sooryavanshi', got %q", last.Title)
	}
}
