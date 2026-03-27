package myskyph_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/myskyph"
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
		ID:      "mysky-8",
		XmltvID: "ABSCBN.ph",
		SiteID:  "8",
		Lang:    "tl",
	}

	date := time.Date(2022, 10, 4, 0, 0, 0, 0, time.UTC)

	programs, err := myskyph.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	if programs[0].Title != "TV PATROL" {
		t.Fatalf("unexpected title: %q", programs[0].Title)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := myskyph.Parse([]byte(`not json`), domain.Channel{}, time.Now())
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseEmpty(t *testing.T) {
	programs, err := myskyph.Parse([]byte(`{"events":[]}`), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
