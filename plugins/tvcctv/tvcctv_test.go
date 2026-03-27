package tvcctv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvcctv"
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
		ID:      "cctv1",
		XmltvID: "CCTV1.cn",
		SiteID:  "cctv1",
		Lang:    "zh",
	}

	programs, err := tvcctv.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title == "" {
		t.Fatal("expected title to be non-empty")
	}

	expectedStart := time.Unix(1701277980, 0).UTC()
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	if p.Source != "tv.cctv.com" {
		t.Fatalf("expected source 'tv.cctv.com', got %q", p.Source)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := tvcctv.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseMissingChannel(t *testing.T) {
	programs, err := tvcctv.Parse([]byte(`{"data":{}}`), domain.Channel{SiteID: "missing"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
