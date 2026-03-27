package i24news_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/i24news"
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
		ID:      "i24news-ar",
		XmltvID: "I24NewsArabic.il",
		SiteID:  "ar",
		Lang:    "ar",
	}

	date := time.Date(2022, 3, 6, 0, 0, 0, 0, time.UTC)

	programs, err := i24news.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]

	expectedStart := time.Date(2022, 3, 6, 13, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 3, 6, 13, 28, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	expectedImage := "https://cdn.i24news.tv/uploads/a1/be/85/20/69/6f/32/1c/ed/b0/f8/5c/f6/1c/40/f9/a1be8520696f321cedb0f85cf61c40f9.png"
	if p.Image != expectedImage {
		t.Fatalf("expected image %q, got %q", expectedImage, p.Image)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := i24news.Parse([]byte(`[]`), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := i24news.Parse([]byte(`not json`), domain.Channel{}, time.Now())
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
