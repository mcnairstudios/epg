package tivutv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tivutv"
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
		ID:      "tivu.tv-62",
		XmltvID: "Rai1HD.it",
		SiteID:  "62",
		Lang:    "it",
	}

	date := time.Date(2022, 10, 4, 0, 0, 0, 0, time.UTC)
	programs, err := tivutv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Cose Nostre - La figlia del boss" {
		t.Fatalf("expected title 'Cose Nostre - La figlia del boss', got %q", p.Title)
	}
	expectedStart := time.Date(2022, 10, 3, 22, 2, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
	expectedStop := time.Date(2022, 10, 3, 22, 45, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	last := programs[len(programs)-1]
	if last.Title != "Tgunomattina - in collaborazione con day" {
		t.Fatalf("expected last title 'Tgunomattina - in collaborazione con day', got %q", last.Title)
	}
	expectedLastStart := time.Date(2022, 10, 5, 4, 58, 0, 0, time.UTC)
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
		ID:      "tivu.tv-62",
		XmltvID: "Rai1HD.it",
		SiteID:  "62",
		Lang:    "it",
	}

	date := time.Date(2022, 10, 4, 0, 0, 0, 0, time.UTC)
	programs, err := tivutv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidHTML(t *testing.T) {
	programs, err := tivutv.Parse([]byte(""), domain.Channel{SiteID: "62"}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
