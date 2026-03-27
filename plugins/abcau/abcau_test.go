package abcau_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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
		SiteID:  "Sydney#ABC1",
		XmltvID: "ABC1.abc.net.au",
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
		t.Fatalf("expected subtitle 'Maggie Beer', got %q", p.SubTitle)
	}

	if p.Image != "https://www.abc.net.au/tv/common/images/publicity/LE1761H002S00_460.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	expectedStart := time.Date(2025, 2, 3, 23, 40, 0, 0, time.FixedZone("AEDT", 11*60*60))
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 2, 4, 0, 9, 0, 0, time.FixedZone("AEDT", 11*60*60))
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "Sydney#ABC1",
		XmltvID: "ABC1.abc.net.au",
		Lang:    "en",
	}

	programs, err := abcau.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := abcau.Parse([]byte(`not json`), domain.Channel{SiteID: "Sydney#ABC1"})
	if err != nil {
		t.Fatalf("expected no error on invalid JSON, got %v", err)
	}
	if programs != nil && len(programs) != 0 {
		t.Fatalf("expected nil/empty programs, got %d", len(programs))
	}
}
