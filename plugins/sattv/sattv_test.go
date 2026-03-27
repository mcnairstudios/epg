package sattv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/sattv"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContentArabic(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_ar.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "sattv-alsaudiya",
		XmltvID: "AlSaudiya.sa",
		SiteID:  "1#38#السعودية",
		Lang:    "ar",
	}

	date := time.Date(2023, 6, 26, 0, 0, 0, 0, time.UTC)
	programs, err := sattv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 35 {
		t.Fatalf("expected 35 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "تعظيم البلد الحرام" {
		t.Fatalf("expected title 'تعظيم البلد الحرام', got %q", p.Title)
	}

	expectedStart := time.Date(2023, 6, 26, 6, 30, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 6, 26, 7, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image != "" {
		t.Fatalf("expected empty image, got %q", p.Image)
	}

	last := programs[34]
	if last.Title != "الأخبار" {
		t.Fatalf("expected last title 'الأخبار', got %q", last.Title)
	}

	expectedLastStart := time.Date(2023, 6, 26, 22, 30, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}

	expectedLastStop := time.Date(2023, 6, 27, 1, 0, 0, 0, time.UTC)
	if !last.Stop.Equal(expectedLastStop) {
		t.Fatalf("expected last stop %v, got %v", expectedLastStop, last.Stop)
	}

	expectedLastImage := "https://sat.tv/wp-content/themes/twentytwenty-child/data_lineups/nilesat/images3/epg-3077892.jpg"
	if last.Image != expectedLastImage {
		t.Fatalf("expected last image %q, got %q", expectedLastImage, last.Image)
	}
}

func TestParseContentEnglish(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_en.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "sattv-saudihd",
		XmltvID: "AlSaudiya.sa",
		SiteID:  "1#38#Saudi HD",
		Lang:    "en",
	}

	date := time.Date(2023, 6, 26, 0, 0, 0, 0, time.UTC)
	programs, err := sattv.Parse(data, channel, date)
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

	expectedStop := time.Date(2023, 6, 26, 10, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID: "1#38#السعودية",
	}

	date := time.Date(2023, 6, 26, 0, 0, 0, 0, time.UTC)
	programs, err := sattv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyContent(t *testing.T) {
	channel := domain.Channel{SiteID: "1#38#test"}
	date := time.Date(2023, 6, 26, 0, 0, 0, 0, time.UTC)

	programs, err := sattv.Parse([]byte(""), channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
