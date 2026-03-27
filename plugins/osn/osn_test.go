package osn_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/osn"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContentArabic(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "osn-FTF",
		XmltvID: "Fatafeat.ae",
		SiteID:  "FTF",
		Lang:    "ar",
	}

	programs, err := osn.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 29 {
		t.Fatalf("expected 29 programs, got %d", len(programs))
	}

	p := programs[1]

	if p.Title != "\u0628\u064a\u062a \u0627\u0644\u062d\u0644\u0648\u064a\u0627\u062a: \u0627\u0644\u062d\u0644\u0642\u0629 3" {
		t.Fatalf("unexpected title: %q", p.Title)
	}

	expectedStart := time.Date(2024, 11, 26, 20, 50, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2024, 11, 26, 21, 45, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseContentEnglish(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "osn-FTF",
		XmltvID: "Fatafeat.ae",
		SiteID:  "FTF",
		Lang:    "en",
	}

	programs, err := osn.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 29 {
		t.Fatalf("expected 29 programs, got %d", len(programs))
	}

	p := programs[1]

	if p.Title != "House Of Desserts: Episode 3" {
		t.Fatalf("expected title 'House Of Desserts: Episode 3', got %q", p.Title)
	}

	expectedStart := time.Date(2024, 11, 26, 20, 50, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2024, 11, 26, 21, 45, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := osn.Parse([]byte(`[]`), domain.Channel{Lang: "ar"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := osn.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatal("expected nil error for invalid JSON (graceful handling)")
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
