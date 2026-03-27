package dishtvIn_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/dishtvIn"
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
		ID:      "dishtv-142639",
		XmltvID: "AndpriveHD.in",
		SiteID:  "142639",
		Lang:    "en",
	}

	programs, err := dishtvIn.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 16 {
		t.Fatalf("expected 16 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Train to Busan 2: Peninsula" {
		t.Fatalf("expected title %q, got %q", "Train to Busan 2: Peninsula", p.Title)
	}

	if p.Image != "https://dtil.tmsimg.com/assets/p17850257_v_h8_am.jpg?lock=1280x720" {
		t.Fatalf("expected image %q, got %q", "https://dtil.tmsimg.com/assets/p17850257_v_h8_am.jpg?lock=1280x720", p.Image)
	}

	expectedStart := time.Date(2025, 1, 25, 19, 0, 0, 0, time.UTC)
	if !p.Start.UTC().Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start.UTC())
	}

	expectedStop := time.Date(2025, 1, 25, 20, 35, 0, 0, time.UTC)
	if !p.Stop.UTC().Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop.UTC())
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := dishtvIn.Parse([]byte("[]"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := dishtvIn.Parse([]byte("not json"), domain.Channel{})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if programs != nil && len(programs) != 0 {
		t.Fatalf("expected nil/empty programs, got %d", len(programs))
	}
}
