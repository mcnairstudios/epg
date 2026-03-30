package tvsmscz_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvsmscz"
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

	ch := domain.Channel{XmltvID: "Test.cz", Lang: "cs", SiteID: "ct1"}
	date := time.Date(2023, 6, 11, 0, 0, 0, 0, time.UTC)
	programs, err := tvsmscz.Parse(data, ch, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	expectedStart := time.Date(2023, 6, 11, 3, 21, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	if p.Title == "" {
		t.Fatal("expected title to be set")
	}

	if p.Source != "m.tv.sms.cz" {
		t.Fatalf("expected source 'm.tv.sms.cz', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	ch := domain.Channel{XmltvID: "Test.cz", Lang: "cs", SiteID: "ct1"}
	programs, err := tvsmscz.Parse(
		[]byte("<!DOCTYPE html><html><head></head><body></body></html>"),
		ch,
		time.Date(2023, 6, 11, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
