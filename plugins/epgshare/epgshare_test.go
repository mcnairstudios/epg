package epgshare_test

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/epgshare"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	f, err := os.Open(testdataPath("content.xml.gz"))
	if err != nil {
		t.Fatalf("failed to open test data: %v", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gz.Close()

	data, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "epgshare01.online-ALJAZEERA1#AlJazeera.English.net",
		XmltvID: "AlJazeera.English.net",
		SiteID:  "ALJAZEERA1#AlJazeera.English.net",
		Lang:    "en",
	}

	date := time.Date(2025, 2, 9, 0, 0, 0, 0, time.UTC)
	programs, err := epgshare.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 40 {
		t.Fatalf("expected 40 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "NEWS 30min" {
		t.Fatalf("expected title 'NEWS 30min', got %q", p.Title)
	}

	expectedStart := time.Date(2025, 2, 8, 23, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 2, 8, 23, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Source != "epgshare01.online" {
		t.Fatalf("expected source 'epgshare01.online', got %q", p.Source)
	}

	last := programs[len(programs)-1]
	if last.Title == "" {
		t.Fatal("expected last program to have a title")
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := epgshare.Parse([]byte{}, domain.Channel{}, time.Time{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidXML(t *testing.T) {
	_, err := epgshare.Parse([]byte("not xml"), domain.Channel{}, time.Time{})
	if err == nil {
		t.Fatal("expected error on invalid XML")
	}
}
