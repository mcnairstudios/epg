package sfrfr_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/sfrfr"
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
		SiteID:  "907",
		XmltvID: "TF1.fr",
		Lang:    "fr",
	}

	programs, err := sfrfr.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	if programs[0].Source != "tv.sfr.fr" {
		t.Fatalf("expected source tv.sfr.fr, got %q", programs[0].Source)
	}
}

func TestParseInvalid(t *testing.T) {
	programs, err := sfrfr.Parse([]byte("not json"), domain.Channel{})
	if err != nil {
		t.Fatalf("should not error: %v", err)
	}
	if programs != nil {
		t.Fatalf("expected nil, got %d", len(programs))
	}
}
