package directvuy_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/directvuy"
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
		SiteID:  "184#VTV",
		XmltvID: "VTV.uy",
		Lang:    "es",
	}

	programs, err := directvuy.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Peñarol vs. Danubio : Fútbol Uruguayo Primera División - Peñarol vs. Danubio" {
		t.Fatalf("unexpected title: %q", p.Title)
	}

	expectedStart := time.Date(2022, 8, 29, 3, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 8, 29, 5, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "VTV.uy" {
		t.Fatalf("expected channel 'VTV.uy', got %q", p.Channel)
	}

	if p.Source != "directv.com.uy" {
		t.Fatalf("expected source 'directv.com.uy', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := directvuy.Parse([]byte(""), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := directvuy.Parse([]byte("not json"), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
