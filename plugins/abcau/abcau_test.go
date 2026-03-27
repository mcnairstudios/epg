package abcau_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/abcau"
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
		ID:      "abcau-ABC1-Sydney",
		XmltvID: "ABC1.au",
		SiteID:  "Sydney#ABC1",
		Lang:    "en",
	}

	programs, err := abcau.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 30 {
		t.Fatalf("expected 30 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Julia Zemiro's Home Delivery" {
		t.Fatalf("expected title %q, got %q", "Julia Zemiro's Home Delivery", p.Title)
	}

	if p.SubTitle != "Maggie Beer" {
		t.Fatalf("expected subtitle %q, got %q", "Maggie Beer", p.SubTitle)
	}

	if p.Image != "https://www.abc.net.au/tv/common/images/publicity/LE1761H002S00_460.jpg" {
		t.Fatalf("expected image %q, got %q", "https://www.abc.net.au/tv/common/images/publicity/LE1761H002S00_460.jpg", p.Image)
	}

	if p.Start.UTC().Format("2006-01-02T15:04:05.000Z") != "2025-02-03T12:40:00.000Z" {
		t.Fatalf("expected start 2025-02-03T12:40:00.000Z, got %v", p.Start.UTC().Format("2006-01-02T15:04:05.000Z"))
	}

	if p.Stop.UTC().Format("2006-01-02T15:04:05.000Z") != "2025-02-03T13:09:00.000Z" {
		t.Fatalf("expected stop 2025-02-03T13:09:00.000Z, got %v", p.Stop.UTC().Format("2006-01-02T15:04:05.000Z"))
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{SiteID: "Sydney#ABC1"}
	programs, err := abcau.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := abcau.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if programs != nil && len(programs) != 0 {
		t.Fatalf("expected nil/empty programs, got %d", len(programs))
	}
}
