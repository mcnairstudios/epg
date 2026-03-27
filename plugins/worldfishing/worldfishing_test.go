package worldfishing_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/worldfishing"
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

	date := time.Date(2023, 1, 24, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		SiteID:  "#",
		XmltvID: "WorldFishingNetwork.us",
		Lang:    "en",
	}

	programs, err := worldfishing.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Major League Fishing" {
		t.Fatalf("expected title 'Major League Fishing', got %q", p.Title)
	}

	if p.SubTitle != "Challenge Cup Sudden Death Round 2" {
		t.Fatalf("expected subtitle 'Challenge Cup Sudden Death Round 2', got %q", p.SubTitle)
	}

	expectedStart := time.Date(2023, 1, 24, 5, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 1, 24, 7, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image != "https://content.osgnetworks.tv/shows/major-league-fishing-thumbnail.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	if p.Channel != "WorldFishingNetwork.us" {
		t.Fatalf("expected channel 'WorldFishingNetwork.us', got %q", p.Channel)
	}

	if p.Source != "worldfishingnetwork.com" {
		t.Fatalf("expected source 'worldfishingnetwork.com', got %q", p.Source)
	}

	if len(programs) < 42 {
		t.Fatalf("expected at least 42 programs, got %d", len(programs))
	}

	last := programs[41]
	if last.Title != "Fishing 411" {
		t.Fatalf("expected last title 'Fishing 411', got %q", last.Title)
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2023, 1, 24, 0, 0, 0, 0, time.UTC)
	programs, err := worldfishing.Parse(data, domain.Channel{}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := worldfishing.Parse([]byte{}, domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
