package tvinsider_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvinsider"
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
		ID:      "tvinsider.com-movieplex",
		XmltvID: "MoviePlexEast.us",
		SiteID:  "movieplex",
		Lang:    "en",
	}

	date := time.Date(2025, 1, 18, 0, 0, 0, 0, time.UTC)
	programs, err := tvinsider.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 15 {
		t.Fatalf("expected 15 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Wild Oats" {
		t.Fatalf("expected title 'Wild Oats', got %q", p.Title)
	}
	if p.Category != "Feature Film" {
		t.Fatalf("expected category 'Feature Film', got %q", p.Category)
	}
	if p.Description != "Two best friends travel to the Canary Islands after one mistakenly receives a large amount of money." {
		t.Fatalf("unexpected description: %q", p.Description)
	}
	expectedStart := time.Date(2025, 1, 18, 5, 45, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
	expectedStop := time.Date(2025, 1, 18, 7, 12, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
	if p.Channel != "MoviePlexEast.us" {
		t.Fatalf("expected channel 'MoviePlexEast.us', got %q", p.Channel)
	}
	if p.Source != "tvinsider.com" {
		t.Fatalf("expected source 'tvinsider.com', got %q", p.Source)
	}

	last := programs[14]
	if last.Title != "Trading Mom" {
		t.Fatalf("expected last title 'Trading Mom', got %q", last.Title)
	}
	expectedLastStart := time.Date(2025, 1, 19, 4, 42, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "tvinsider.com-movieplex",
		XmltvID: "MoviePlexEast.us",
		SiteID:  "movieplex",
		Lang:    "en",
	}

	date := time.Date(2025, 1, 18, 0, 0, 0, 0, time.UTC)
	programs, err := tvinsider.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidHTML(t *testing.T) {
	programs, err := tvinsider.Parse([]byte(""), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
