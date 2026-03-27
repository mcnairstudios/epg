package kplusvn_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/kplusvn"
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
		XmltvID: "KPlusKids.vn",
		SiteID:  "F31WvdXdwYNOInUaTeIC-ixsLQVsrSNgUczDSFCN",
		Lang:    "vi",
	}

	date := time.Date(2025, 1, 18, 0, 0, 0, 0, time.UTC)
	programs, err := kplusvn.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 100 {
		t.Fatalf("expected 100 programs, got %d", len(programs))
	}

	if programs[0].Title != "Masha and the Bear S1, Ep01" {
		t.Fatalf("expected title 'Masha and the Bear S1, Ep01', got %q", programs[0].Title)
	}

	expectedStart := time.Date(2025, 1, 18, 0, 3, 0, 0, time.UTC)
	if !programs[0].Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, programs[0].Start)
	}

	expectedStop := time.Date(2025, 1, 18, 0, 10, 0, 0, time.UTC)
	if !programs[0].Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, programs[0].Stop)
	}

	if programs[0].Category != "Children" {
		t.Fatalf("expected category 'Children', got %q", programs[0].Category)
	}

	if programs[0].Image == "" {
		t.Fatal("expected image to be set")
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := kplusvn.Parse(data, domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
