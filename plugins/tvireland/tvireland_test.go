package tvireland_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvireland"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseResponse(t *testing.T) {
	content, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "2378/virgin-media-more",
		XmltvID: "VirginMediaMore.ie",
	}

	date := time.Date(2023, 11, 25, 0, 0, 0, 0, time.UTC)

	programs, err := tvireland.Parse(content, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 14 {
		t.Fatalf("expected 14 programs, got %d", len(programs))
	}

	p0 := programs[0]
	if p0.Title != "Best of Rugby World Cup" {
		t.Fatalf("expected title 'Best of Rugby World Cup', got %q", p0.Title)
	}

	expectedStart0 := "2023-11-25T00:20:00Z"
	if p0.Start.UTC().Format("2006-01-02T15:04:05Z") != expectedStart0 {
		t.Fatalf("expected start %s, got %s", expectedStart0, p0.Start.UTC().Format("2006-01-02T15:04:05Z"))
	}

	expectedStop0 := "2023-11-25T01:10:00Z"
	if p0.Stop.UTC().Format("2006-01-02T15:04:05Z") != expectedStop0 {
		t.Fatalf("expected stop %s, got %s", expectedStop0, p0.Stop.UTC().Format("2006-01-02T15:04:05Z"))
	}

	p13 := programs[13]
	if p13.Title != "Stories from the Street" {
		t.Fatalf("expected title 'Stories from the Street', got %q", p13.Title)
	}

	expectedStart13 := "2023-11-25T23:30:00Z"
	if p13.Start.UTC().Format("2006-01-02T15:04:05Z") != expectedStart13 {
		t.Fatalf("expected start %s, got %s", expectedStart13, p13.Start.UTC().Format("2006-01-02T15:04:05Z"))
	}

	expectedStop13 := "2023-11-26T00:00:00Z"
	if p13.Stop.UTC().Format("2006-01-02T15:04:05Z") != expectedStop13 {
		t.Fatalf("expected stop %s, got %s", expectedStop13, p13.Stop.UTC().Format("2006-01-02T15:04:05Z"))
	}

	if p0.Channel != "VirginMediaMore.ie" {
		t.Fatalf("expected channel 'VirginMediaMore.ie', got %q", p0.Channel)
	}

	if p0.Source != "tvireland.ie" {
		t.Fatalf("expected source 'tvireland.ie', got %q", p0.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	content := []byte("<!DOCTYPE html><html><head></head><body></body></html>")

	channel := domain.Channel{
		SiteID:  "2378/virgin-media-more",
		XmltvID: "VirginMediaMore.ie",
	}

	date := time.Date(2023, 11, 25, 0, 0, 0, 0, time.UTC)

	programs, err := tvireland.Parse(content, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
