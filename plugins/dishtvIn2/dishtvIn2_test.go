package dishtvIn2_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/dishtvIn2"
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
		ID:      "dishtv.in-142639",
		XmltvID: "AndpriveHD.in",
		SiteID:  "142639",
		Lang:    "en",
	}

	programs, err := dishtvIn2.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 16 {
		t.Fatalf("expected 16 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Train to Busan 2: Peninsula" {
		t.Fatalf("expected title 'Train to Busan 2: Peninsula', got %q", p.Title)
	}

	if p.Description != "Jung Seok, a former soldier, along with his teammates, sets out on a mission to battle hordes of post-apocalyptic zombies in the Korean peninsula wastelands." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	if p.Category != "Film" {
		t.Fatalf("expected category 'Film', got %q", p.Category)
	}

	expectedStart := time.Date(2025, 1, 25, 19, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 1, 25, 20, 35, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image != "https://dtil.tmsimg.com/assets/p17850257_v_h8_am.jpg?lock=1280x720" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	if p.Channel != "AndpriveHD.in" {
		t.Fatalf("expected channel 'AndpriveHD.in', got %q", p.Channel)
	}

	if p.Source != "dishtv.in" {
		t.Fatalf("expected source 'dishtv.in', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := dishtvIn2.Parse([]byte(`[]`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := dishtvIn2.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseEpisodeNumber(t *testing.T) {
	data := []byte(`[{"title":"Test","start":"2025-01-26T00:30:00Z","stop":"2025-01-26T01:30:00Z","episode-num":"5","regional":{},"images":{},"credits":{}}]`)
	programs, err := dishtvIn2.Parse(data, domain.Channel{Lang: "en"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}
	if programs[0].Episode != 5 {
		t.Fatalf("expected episode 5, got %d", programs[0].Episode)
	}
}
