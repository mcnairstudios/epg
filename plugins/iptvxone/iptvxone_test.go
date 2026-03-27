package iptvxone_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/iptvxone"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.xml"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "epg.iptvx.one-12-omsk",
		XmltvID: "Channel12.ru",
		SiteID:  "12-omsk",
		Lang:    "ru",
	}

	date := time.Date(2025, 9, 24, 0, 0, 0, 0, time.UTC)
	programs, err := iptvxone.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Час новостей" {
		t.Fatalf("expected title 'Час новостей', got %q", p.Title)
	}

	expectedStart := time.Date(2025, 9, 24, 0, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 9, 24, 0, 25, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "Channel12.ru" {
		t.Fatalf("expected channel 'Channel12.ru', got %q", p.Channel)
	}

	if p.Source != "epg.iptvx.one" {
		t.Fatalf("expected source 'epg.iptvx.one', got %q", p.Source)
	}
}

func TestParseGzippedContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.xml.gz"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "epg.iptvx.one-12-omsk",
		XmltvID: "Channel12.ru",
		SiteID:  "12-omsk",
		Lang:    "ru",
	}

	date := time.Date(2025, 9, 24, 0, 0, 0, 0, time.UTC)
	programs, err := iptvxone.ParseGzipped(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program from gzipped data")
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := iptvxone.Parse([]byte(``), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
