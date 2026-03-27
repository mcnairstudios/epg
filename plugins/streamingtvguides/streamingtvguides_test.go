package streamingtvguides_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/streamingtvguides"
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

	channel := domain.Channel{
		SiteID:  "GMAPNY",
		XmltvID: "GMAPinoyTVUSACanada.ph",
		Lang:    "en",
	}

	date := time.Date(2023, 6, 27, 0, 0, 0, 0, time.UTC)
	programs, err := streamingtvguides.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 38 {
		t.Fatalf("expected 38 programs, got %d", len(programs))
	}

	first := programs[0]
	if first.Title != "24 Oras" {
		t.Fatalf("expected title '24 Oras', got %q", first.Title)
	}
	if first.Description != "Up to the minute news around the world." {
		t.Fatalf("expected description 'Up to the minute news around the world.', got %q", first.Description)
	}
	expectedStart := time.Date(2023, 6, 27, 0, 40, 0, 0, time.UTC)
	if !first.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, first.Start)
	}
	expectedStop := time.Date(2023, 6, 27, 2, 0, 0, 0, time.UTC)
	if !first.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, first.Stop)
	}
	if first.Channel != "GMAPinoyTVUSACanada.ph" {
		t.Fatalf("expected channel 'GMAPinoyTVUSACanada.ph', got %q", first.Channel)
	}
	if first.Source != "streamingtvguides.com" {
		t.Fatalf("expected source 'streamingtvguides.com', got %q", first.Source)
	}

	last := programs[37]
	if last.Title != "Eat Bulaga" {
		t.Fatalf("expected title 'Eat Bulaga', got %q", last.Title)
	}
	if last.Description != "Rousing and engrossing segments with engaging hosts." {
		t.Fatalf("expected description 'Rousing and engrossing segments with engaging hosts.', got %q", last.Description)
	}
	expectedLastStart := time.Date(2023, 6, 27, 21, 50, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected start %v, got %v", expectedLastStart, last.Start)
	}
	expectedLastStop := time.Date(2023, 6, 28, 0, 0, 0, 0, time.UTC)
	if !last.Stop.Equal(expectedLastStop) {
		t.Fatalf("expected stop %v, got %v", expectedLastStop, last.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "GMAPNY",
		XmltvID: "GMAPinoyTVUSACanada.ph",
		Lang:    "en",
	}

	date := time.Date(2023, 6, 27, 0, 0, 0, 0, time.UTC)
	programs, err := streamingtvguides.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidHTML(t *testing.T) {
	programs, err := streamingtvguides.Parse([]byte("not html"), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
