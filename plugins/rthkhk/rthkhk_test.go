package rthkhk_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/rthkhk"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContentZH(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_zh.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		XmltvID: "RTHKTV31.hk",
		SiteID:  "31",
		Lang:    "zh",
	}

	date := time.Date(2022, 12, 2, 0, 0, 0, 0, time.UTC)
	programs, err := rthkhk.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	expectedStart := time.Date(2022, 12, 1, 16, 0, 0, 0, time.UTC)
	if !programs[0].Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, programs[0].Start)
	}

	expectedStop := time.Date(2022, 12, 1, 17, 0, 0, 0, time.UTC)
	if !programs[0].Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, programs[0].Stop)
	}

	if programs[0].Image != "https://www.rthk.hk/assets/images/rthk/dtt31/thegreataerospace/10239_1920_s.jpg" {
		t.Fatalf("unexpected image: %q", programs[0].Image)
	}
}

func TestParseContentEN(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_en.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		XmltvID: "RTHKTV31.hk",
		SiteID:  "31",
		Lang:    "en",
	}

	date := time.Date(2022, 12, 2, 0, 0, 0, 0, time.UTC)
	programs, err := rthkhk.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	if programs[0].Title != "The Great Aerospace" {
		t.Fatalf("expected title 'The Great Aerospace', got %q", programs[0].Title)
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := rthkhk.Parse(data, domain.Channel{SiteID: "31"}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
