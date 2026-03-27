package tapdmv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tapdmv"
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
		ID:      "tapdmv-1",
		XmltvID: "TapDMV.us",
		SiteID:  "1",
		Lang:    "en",
	}

	programs, err := tapdmv.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "The Bourne Ultimatum" {
		t.Fatalf("unexpected title: %q", p.Title)
	}

	if p.Source != "tapdmv.com" {
		t.Fatalf("expected source 'tapdmv.com', got %q", p.Source)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := tapdmv.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
