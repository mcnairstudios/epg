package getafteritmedia_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/getafteritmedia"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

var channel = domain.Channel{
	ID:      "getafteritmedia.com-494637005",
	XmltvID: "REVNWebFeed.us",
	SiteID:  "494637005",
	Lang:    "en",
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2022, 11, 26, 0, 0, 0, 0, time.UTC)
	programs, err := getafteritmedia.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "The Appraisers" {
		t.Fatalf("expected title 'The Appraisers', got %q", p.Title)
	}

	expectedStart := time.Date(2022, 11, 26, 5, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 11, 26, 5, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "REVNWebFeed.us" {
		t.Fatalf("expected channel 'REVNWebFeed.us', got %q", p.Channel)
	}

	if p.Source != "getafteritmedia.com" {
		t.Fatalf("expected source 'getafteritmedia.com', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := getafteritmedia.Parse([]byte("<!DOCTYPE html><html><head></head><body></body></html>"), channel, time.Date(2022, 11, 26, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
