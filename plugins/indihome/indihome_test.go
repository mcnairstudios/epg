package indihome_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/indihome"
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

	date := time.Date(2022, 8, 8, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "indihome-metrotv",
		XmltvID: "MetroTV.id",
		SiteID:  "metrotv",
		Lang:    "id",
	}

	programs, err := indihome.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 4 {
		t.Fatalf("expected 4 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Headline News" {
		t.Fatalf("expected title 'Headline News', got %q", p.Title)
	}
	expectedStart := time.Date(2022, 8, 8, 0, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
	expectedStop := time.Date(2022, 8, 8, 0, 5, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	p1 := programs[1]
	if p1.Title != "Editorial Media Indonesia" {
		t.Fatalf("expected title 'Editorial Media Indonesia', got %q", p1.Title)
	}
	expectedStart1 := time.Date(2022, 8, 8, 0, 5, 0, 0, time.UTC)
	if !p1.Start.Equal(expectedStart1) {
		t.Fatalf("expected start %v, got %v", expectedStart1, p1.Start)
	}
	expectedStop1 := time.Date(2022, 8, 8, 0, 30, 0, 0, time.UTC)
	if !p1.Stop.Equal(expectedStop1) {
		t.Fatalf("expected stop %v, got %v", expectedStop1, p1.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2022, 8, 8, 0, 0, 0, 0, time.UTC)
	programs, err := indihome.Parse(data, domain.Channel{}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
