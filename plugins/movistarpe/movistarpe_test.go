package movistarpe_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/movistarpe"
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
		ID:      "movistarpe-1",
		XmltvID: "MovistarPE.pe",
		SiteID:  "lch2219",
		Lang:    "es",
	}

	programs, err := movistarpe.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	if programs[0].Title == "" {
		t.Fatal("expected non-empty title")
	}

	if programs[0].Source != "tv.movistar.com.pe" {
		t.Fatalf("expected source 'tv.movistar.com.pe', got %q", programs[0].Source)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := movistarpe.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseEmpty(t *testing.T) {
	programs, err := movistarpe.Parse([]byte(`{"Content":[]}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
