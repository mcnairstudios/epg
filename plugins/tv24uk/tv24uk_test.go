package tv24uk_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tv24uk"
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

	date := time.Date(2022, 8, 28, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "tv24uk-bbc-two",
		XmltvID: "BBCTwo.uk",
		SiteID:  "bbc-two",
		Lang:    "en",
	}

	programs, err := tv24uk.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Gardeners' World" {
		t.Fatalf("expected title \"Gardeners' World\", got %q", p.Title)
	}
	expectedStart := time.Date(2022, 8, 28, 5, 5, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
	expectedStop := time.Date(2022, 8, 28, 6, 5, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	p22 := programs[22]
	if p22.Title != "Animal Park" {
		t.Fatalf("expected title 'Animal Park', got %q", p22.Title)
	}
	expectedStart22 := time.Date(2022, 8, 29, 5, 30, 0, 0, time.UTC)
	if !p22.Start.Equal(expectedStart22) {
		t.Fatalf("expected start %v, got %v", expectedStart22, p22.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := tv24uk.Parse([]byte(``), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
