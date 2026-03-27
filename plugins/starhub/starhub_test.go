package starhub_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/starhub"
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
		ID:      "starhub-d258444e-b66b-4cbe-88db-e09f31ab8a1f",
		XmltvID: "AXN.sg",
		SiteID:  "d258444e-b66b-4cbe-88db-e09f31ab8a1f",
		Lang:    "en",
	}

	programs, err := starhub.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Northern Rexposure" {
		t.Fatalf("expected title 'Northern Rexposure', got %q", p.Title)
	}

	if p.SubTitle != "Hudson & Rex (Season 5)" {
		t.Fatalf("expected subtitle 'Hudson & Rex (Season 5)', got %q", p.SubTitle)
	}

	if p.Description != "When Jesse's sister contacts him for help, he, Sarah and Rex head to Northern Ontario and find themselves in the middle of a deadly situation." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	expectedStart := time.Date(2024, 12, 3, 17, 25, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2024, 12, 3, 18, 20, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Season != 5 {
		t.Fatalf("expected season 5, got %d", p.Season)
	}

	if p.Episode != 15 {
		t.Fatalf("expected episode 15, got %d", p.Episode)
	}

	if p.Category != "Drama" {
		t.Fatalf("expected category 'Drama', got %q", p.Category)
	}

	if p.Image != "https://poster.starhubgo.com/poster/ch511_hudson_rex5.jpg?w=960&h=540" {
		t.Fatalf("expected image URL, got %q", p.Image)
	}

	if p.Channel != "AXN.sg" {
		t.Fatalf("expected channel 'AXN.sg', got %q", p.Channel)
	}

	if p.Source != "starhubtvplus.com" {
		t.Fatalf("expected source 'starhubtvplus.com', got %q", p.Source)
	}
}

func TestParseEmptyResources(t *testing.T) {
	programs, err := starhub.Parse([]byte(`{"resources":[],"page":{"current":1,"total":1}}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := starhub.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseNoResources(t *testing.T) {
	programs, err := starhub.Parse([]byte(`{}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseSeasonFromSTitle(t *testing.T) {
	data := []byte(`{"resources":[{"title":"Test","serie_title":"Show S3","start":1000,"end":2000,"genres":[],"pictures":[]}],"page":{"current":1,"total":1}}`)
	programs, err := starhub.Parse(data, domain.Channel{Lang: "en"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}
	if programs[0].Season != 3 {
		t.Fatalf("expected season 3, got %d", programs[0].Season)
	}
}

func TestParseNoSeason(t *testing.T) {
	data := []byte(`{"resources":[{"title":"Test","serie_title":"No season here","start":1000,"end":2000,"genres":[],"pictures":[]}],"page":{"current":1,"total":1}}`)
	programs, err := starhub.Parse(data, domain.Channel{Lang: "en"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if programs[0].Season != 0 {
		t.Fatalf("expected season 0, got %d", programs[0].Season)
	}
}
