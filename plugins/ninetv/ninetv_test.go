package ninetv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/ninetv"
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

	date := time.Date(2022, 3, 6, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		SiteID:  "#",
		XmltvID: "Channel9.il",
		Lang:    "ru",
	}

	programs, err := ninetv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) < 2 {
		t.Fatalf("expected at least 2 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "\u0421\u043b\u0435\u043f\u0430\u044f" {
		t.Fatalf("unexpected title: %q", p.Title)
	}

	expectedStart := time.Date(2022, 3, 6, 4, 30, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 3, 6, 7, 10, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image != "https://www.9tv.co.il/download/pictures/img_id=8484.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	p1 := programs[1]
	expectedStart = time.Date(2022, 3, 6, 7, 10, 0, 0, time.UTC)
	if !p1.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p1.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2022, 3, 6, 0, 0, 0, 0, time.UTC)
	programs, err := ninetv.Parse(data, domain.Channel{}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
