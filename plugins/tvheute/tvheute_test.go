package tvheute_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvheute"
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
		ID:      "tvheute-orf1",
		XmltvID: "ORF1.at",
		SiteID:  "orf1",
		Lang:    "de",
	}

	programs, err := tvheute.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Monchhichi (Wh.)" {
		t.Fatalf("expected title 'Monchhichi (Wh.)', got %q", p.Title)
	}
	if p.Category != "Kids" {
		t.Fatalf("expected category 'Kids', got %q", p.Category)
	}

	expectedStart := time.Date(2021, 11, 8, 5, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
	expectedStop := time.Date(2021, 11, 8, 5, 10, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image != "https://tvheute.at/images/orf1/monchhichi_kids--1895216560-00.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	if !containsSubstring(p.Description, "Roger hat sich") {
		t.Fatalf("expected description to contain 'Roger hat sich', got %q", p.Description)
	}

	p1 := programs[1]
	if p1.Title != "ZIB 18" {
		t.Fatalf("expected title 'ZIB 18', got %q", p1.Title)
	}
	expectedStart1 := time.Date(2021, 11, 8, 17, 0, 0, 0, time.UTC)
	if !p1.Start.Equal(expectedStart1) {
		t.Fatalf("expected start %v, got %v", expectedStart1, p1.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := tvheute.Parse(data, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
