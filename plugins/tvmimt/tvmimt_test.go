package tvmimt_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvmimt"
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

	date := time.Date(2022, 10, 29, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "tvmimt-2",
		XmltvID: "TVM.mt",
		SiteID:  "2",
		Lang:    "mt",
	}

	programs, err := tvmimt.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) < 2 {
		t.Fatalf("expected at least 2 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Bizzilla" {
		t.Fatalf("expected title 'Bizzilla', got %q", p.Title)
	}

	expectedStart := time.Date(2022, 10, 29, 3, 30, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 10, 29, 4, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image == "" {
		t.Fatal("expected image to be set")
	}

	p1 := programs[1]
	if p1.Title != "The Adventures of Puss in Boots" {
		t.Fatalf("expected second title 'The Adventures of Puss in Boots', got %q", p1.Title)
	}

	expectedStart1 := time.Date(2022, 10, 29, 4, 0, 0, 0, time.UTC)
	if !p1.Start.Equal(expectedStart1) {
		t.Fatalf("expected second start %v, got %v", expectedStart1, p1.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	date := time.Date(2022, 10, 29, 0, 0, 0, 0, time.UTC)
	programs, err := tvmimt.Parse(
		[]byte("<!doctype html><html><head></head><body></body></html>"),
		domain.Channel{},
		date,
	)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
