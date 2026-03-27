package nhkjp_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/nhkjp"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("schedule.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "0",
		XmltvID: "NHKWorldJapan.jp",
		Lang:    "en",
	}

	programs, err := nhkjp.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "NHK NEWSLINE" {
		t.Fatalf("expected title 'NHK NEWSLINE', got %q", p.Title)
	}
	if p.SubTitle != "" {
		t.Fatalf("expected empty subtitle, got %q", p.SubTitle)
	}

	expectedStart := time.Date(2025, 10, 19, 15, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 10, 19, 15, 10, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	p1 := programs[1]
	if p1.Title != "J-MELO" {
		t.Fatalf("expected title 'J-MELO', got %q", p1.Title)
	}
	if p1.SubTitle != "Furui Riho and shallm" {
		t.Fatalf("expected subtitle 'Furui Riho and shallm', got %q", p1.SubTitle)
	}
	if p1.Image == "" {
		t.Fatal("expected image to be set for second program")
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := nhkjp.Parse([]byte(""), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := nhkjp.Parse([]byte("not json"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
