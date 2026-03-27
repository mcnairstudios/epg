package orangees_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/orangees"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data1, err := os.ReadFile(testdataPath("data1.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}
	data2, err := os.ReadFile(testdataPath("data2.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}
	data3, err := os.ReadFile(testdataPath("data3.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "orangees-1010",
		XmltvID: "La1.es",
		SiteID:  "1010",
		Lang:    "es",
	}

	programs, err := orangees.ParseMulti([][]byte{data1, data2, data3}, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 18 {
		t.Fatalf("expected 18 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Una joven prometedora" {
		t.Fatalf("expected title 'Una joven prometedora', got %q", p.Title)
	}
	if p.Start.Format("2006-01-02T15:04:05.000Z") != "2025-01-11T22:55:00.000Z" {
		t.Fatalf("unexpected start: %v", p.Start)
	}
	if p.Stop.Format("2006-01-02T15:04:05.000Z") != "2025-01-12T00:40:00.000Z" {
		t.Fatalf("unexpected stop: %v", p.Stop)
	}
	if p.Category != "Cine, Drama, Suspense" {
		t.Fatalf("expected category 'Cine, Drama, Suspense', got %q", p.Category)
	}
	if p.Image != "https://pc.orangetv.orange.es/pc/api/rtv/v1/images/epg/COVER/COVER_2247567.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	last := programs[17]
	if last.Title != "Bake Off: Famosos al horno - T2, E01: Bake Off: Famosos al horno" {
		t.Fatalf("expected last title, got %q", last.Title)
	}
	if last.Season != 2 {
		t.Fatalf("expected season 2, got %d", last.Season)
	}
	if last.Episode != 1 {
		t.Fatalf("expected episode 1, got %d", last.Episode)
	}
}

func TestParseSingleSegment(t *testing.T) {
	data, err := os.ReadFile(testdataPath("data1.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "orangees-1010",
		XmltvID: "La1.es",
		SiteID:  "1010",
		Lang:    "es",
	}

	programs, err := orangees.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 4 {
		t.Fatalf("expected 4 programs, got %d", len(programs))
	}
}

func TestParseEmptyContent(t *testing.T) {
	channel := domain.Channel{SiteID: "1010"}
	programs, err := orangees.Parse([]byte(`{}`), channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	channel := domain.Channel{SiteID: "1010"}
	programs, err := orangees.Parse([]byte(`not json`), channel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
