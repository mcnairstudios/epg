package skyuk_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/skyuk"
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
		ID:      "sky-4086",
		XmltvID: "SkyHistoryHD.uk",
		SiteID:  "4086",
		Lang:    "en",
	}

	date := time.Date(2024, 12, 14, 0, 0, 0, 0, time.UTC)

	programs, err := skyuk.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 31 {
		t.Fatalf("expected 31 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Storage Wars" {
		t.Fatalf("expected title 'Storage Wars', got %q", p.Title)
	}

	if p.Description != "A Sale Of Two Cities: Emily brings her mother along with her to Walnut, and Darrell wastes no time finding an advantage. Ivy and Ivy jr clean up with their locker. (S12, ep 4)" {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	if p.Season != 12 {
		t.Fatalf("expected season 12, got %d", p.Season)
	}

	if p.Episode != 4 {
		t.Fatalf("expected episode 4, got %d", p.Episode)
	}

	expectedStart := time.Date(2024, 12, 14, 0, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2024, 12, 14, 0, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image != "https://images.metadata.sky.com/pd-image/b9572a38-8db7-471e-a2d7-462e1dd26af2/16-9/640" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	if p.Channel != "SkyHistoryHD.uk" {
		t.Fatalf("expected channel 'SkyHistoryHD.uk', got %q", p.Channel)
	}

	if p.Source != "sky.com" {
		t.Fatalf("expected source 'sky.com', got %q", p.Source)
	}

	if p.Lang != "en" {
		t.Fatalf("expected lang 'en', got %q", p.Lang)
	}

	p2 := programs[2]

	if p2.Title != "Storage Wars" {
		t.Fatalf("expected title 'Storage Wars', got %q", p2.Title)
	}

	if p2.Description != "Not All That Glitters Is Gourd: Back in the city of Orange, the Vegas Ladies arrive in vintage style - though not everyone agrees. (S12, ep 6)" {
		t.Fatalf("unexpected description for program[2]: %q", p2.Description)
	}

	if p2.Season != 12 {
		t.Fatalf("expected season 12, got %d", p2.Season)
	}

	if p2.Episode != 6 {
		t.Fatalf("expected episode 6, got %d", p2.Episode)
	}

	expectedStart2 := time.Date(2024, 12, 14, 1, 0, 0, 0, time.UTC)
	if !p2.Start.Equal(expectedStart2) {
		t.Fatalf("expected start %v, got %v", expectedStart2, p2.Start)
	}

	expectedStop2 := time.Date(2024, 12, 14, 1, 30, 0, 0, time.UTC)
	if !p2.Stop.Equal(expectedStop2) {
		t.Fatalf("expected stop %v, got %v", expectedStop2, p2.Stop)
	}

	if p2.Image != "https://images.metadata.sky.com/pd-image/e9521ccc-bdcc-4075-9c2e-bc835247148b/16-9/640" {
		t.Fatalf("unexpected image for program[2]: %q", p2.Image)
	}
}

func TestParseEmptyContent(t *testing.T) {
	channel := domain.Channel{SiteID: "4086"}
	date := time.Date(2024, 12, 14, 0, 0, 0, 0, time.UTC)

	programs, err := skyuk.Parse([]byte(""), channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	channel := domain.Channel{SiteID: "4086"}
	date := time.Date(2024, 12, 14, 0, 0, 0, 0, time.UTC)

	_, err := skyuk.Parse([]byte("not json"), channel, date)
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseNoSchedule(t *testing.T) {
	channel := domain.Channel{SiteID: "4086"}
	date := time.Date(2024, 12, 14, 0, 0, 0, 0, time.UTC)

	programs, err := skyuk.Parse([]byte(`{}`), channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
