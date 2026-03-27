package mediagenie_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/mediagenie"
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

	date := time.Date(2023, 1, 25, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		SiteID:  "ENA_DRAMA",
		XmltvID: "ENADRAMA.kr",
		Lang:    "ko",
	}

	programs, err := mediagenie.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "대행사" {
		t.Fatalf("expected title '대행사', got %q", p.Title)
	}

	expectedStart := time.Date(2023, 1, 24, 15, 20, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 1, 24, 16, 34, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "ENADRAMA.kr" {
		t.Fatalf("expected channel 'ENADRAMA.kr', got %q", p.Channel)
	}

	if p.Source != "mediagenie.co.kr" {
		t.Fatalf("expected source 'mediagenie.co.kr', got %q", p.Source)
	}

	if len(programs) < 17 {
		t.Fatalf("expected at least 17 programs, got %d", len(programs))
	}

	last := programs[16]
	if last.Title != "법쩐" {
		t.Fatalf("expected program 17 title '법쩐', got %q", last.Title)
	}

	expectedLastStart := time.Date(2023, 1, 25, 14, 27, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected program 17 start %v, got %v", expectedLastStart, last.Start)
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2023, 1, 25, 0, 0, 0, 0, time.UTC)
	programs, err := mediagenie.Parse(data, domain.Channel{}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := mediagenie.Parse([]byte{}, domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
