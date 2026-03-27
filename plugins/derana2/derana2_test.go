package derana2_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/derana2"
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
		ID:      "derana.lk.v2-#",
		XmltvID: "TVDerana.lk",
		SiteID:  "#",
		Lang:    "sn",
	}

	programs, err := derana2.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	if programs[0].Title == "" {
		t.Fatal("expected non-empty title")
	}

	if programs[0].Source != "derana.lk.v2" {
		t.Fatalf("expected source 'derana.lk.v2', got %q", programs[0].Source)
	}

	if programs[0].Start.IsZero() {
		t.Fatal("expected non-zero start time")
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := derana2.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseEmptySchedule(t *testing.T) {
	programs, err := derana2.Parse([]byte(`{"all_schedules":[]}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
