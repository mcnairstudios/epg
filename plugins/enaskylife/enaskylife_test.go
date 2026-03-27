package enaskylife_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/enaskylife"
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
		ID:      "ena.skylifetv.co.kr-ENA",
		XmltvID: "ENA.kr",
		SiteID:  "ENA",
		Lang:    "ko",
	}

	date := time.Date(2023, 1, 27, 0, 0, 0, 0, time.UTC)
	programs, err := enaskylife.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "법쩐 6화" {
		t.Fatalf("expected title '법쩐 6화', got %q", p.Title)
	}

	expectedStart := time.Date(2023, 1, 26, 16, 5, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 1, 26, 17, 20, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Source != "ena.skylifetv.co.kr" {
		t.Fatalf("expected source 'ena.skylifetv.co.kr', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{SiteID: "ENA"}
	date := time.Date(2023, 1, 27, 0, 0, 0, 0, time.UTC)

	programs, err := enaskylife.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse should not fail: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNoData(t *testing.T) {
	programs, err := enaskylife.Parse([]byte(""), domain.Channel{}, time.Time{})
	if err != nil {
		t.Fatalf("parse should not fail: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
