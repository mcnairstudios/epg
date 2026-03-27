package dnafi_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/dnafi"
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
		ID:      "dna-mtv3",
		XmltvID: "MTV3.fi",
		SiteID:  "ch-216356",
		Lang:    "fi",
	}

	date := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	programs, err := dnafi.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 20 {
		t.Fatalf("expected 20 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Next Level Chef" {
		t.Fatalf("unexpected title: %q", p.Title)
	}

	if p.SubTitle != "Brunssi" {
		t.Fatalf("unexpected subtitle: %q", p.SubTitle)
	}

	if p.Season != 1 {
		t.Fatalf("expected season 1, got %d", p.Season)
	}

	if p.Episode != 6 {
		t.Fatalf("expected episode 6, got %d", p.Episode)
	}

	expectedStart := time.Date(2025, 1, 15, 2, 30, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 1, 15, 3, 22, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := dnafi.Parse([]byte(""), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := dnafi.Parse([]byte("not json"), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse should not error, got: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
