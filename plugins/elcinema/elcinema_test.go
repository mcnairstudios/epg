package elcinema_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/elcinema"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContentEN(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.en.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2022, 8, 28, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "elcinema-1254",
		XmltvID: "OSNSeries.ae",
		SiteID:  "1254",
		Lang:    "en",
	}

	programs, err := elcinema.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Station 19 S5" {
		t.Fatalf("expected title 'Station 19 S5', got %q", p.Title)
	}
	if p.Image != "https://media.elcinema.com/uploads/_150x200_ec30d1a2251c8edf83334be4860184c74d2534d7ba508a334ad66fa59acc4926.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}
	if p.Category != "Series" {
		t.Fatalf("expected category 'Series', got %q", p.Category)
	}

	expectedStart := time.Date(2022, 8, 28, 2, 25, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
	expectedStop := time.Date(2022, 8, 28, 3, 15, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := elcinema.Parse(
		[]byte(`<!DOCTYPE html><html lang="ar" dir="rtl"><head></head><body></body></html>`),
		domain.Channel{Lang: "en"},
		time.Date(2022, 8, 28, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
