package sattv2_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/sattv2"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseArabicContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_ar.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2023, 6, 26, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		SiteID:  "1#38#\u0627\u0644\u0633\u0639\u0648\u062f\u064a\u0629",
		XmltvID: "AlSaudiya.sa",
		Lang:    "ar",
	}

	programs, err := sattv2.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 35 {
		t.Fatalf("expected 35 programs, got %d", len(programs))
	}

	p := programs[0]
	expectedStart := time.Date(2023, 6, 26, 6, 30, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 6, 26, 7, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	last := programs[34]
	expectedStart = time.Date(2023, 6, 26, 22, 30, 0, 0, time.UTC)
	if !last.Start.Equal(expectedStart) {
		t.Fatalf("expected last start %v, got %v", expectedStart, last.Start)
	}

	expectedStop = time.Date(2023, 6, 27, 1, 0, 0, 0, time.UTC)
	if !last.Stop.Equal(expectedStop) {
		t.Fatalf("expected last stop %v, got %v", expectedStop, last.Stop)
	}
}

func TestParseEnglishContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_en.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2023, 6, 26, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		SiteID:  "1#38#Saudi HD",
		XmltvID: "AlSaudiya.sa",
		Lang:    "en",
	}

	programs, err := sattv2.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 32 {
		t.Fatalf("expected 32 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "News" {
		t.Fatalf("expected title 'News', got %q", p.Title)
	}

	expectedStart := time.Date(2023, 6, 26, 9, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2023, 6, 26, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		SiteID:  "1#38#\u0627\u0644\u0633\u0639\u0648\u062f\u064a\u0629",
		XmltvID: "AlSaudiya.sa",
		Lang:    "ar",
	}

	programs, err := sattv2.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
