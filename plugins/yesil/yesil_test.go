package yesil_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/yesil"
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
		ID:      "yes.co.il-YSA1",
		XmltvID: "YesCinemaAction.il",
		SiteID:  "YSA1",
		Lang:    "he",
	}

	programs, err := yesil.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 13 {
		t.Fatalf("expected 13 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "הבית" {
		t.Fatalf("expected title 'הבית', got %q", p.Title)
	}

	expectedStart := time.Date(2025, 1, 29, 23, 52, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 1, 30, 1, 31, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image != "https://fykswkmjb.filerobot.com/VodAndHomeChan/VP001212610.JPG" {
		t.Fatalf("expected image URL, got %q", p.Image)
	}

	if p.Channel != "YesCinemaAction.il" {
		t.Fatalf("expected channel 'YesCinemaAction.il', got %q", p.Channel)
	}

	if p.Source != "yes.co.il" {
		t.Fatalf("expected source 'yes.co.il', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := yesil.Parse([]byte(`{"items":[]}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := yesil.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
