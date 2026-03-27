package vrtbe_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/vrtbe"
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
		ID:      "vrt-vrt1",
		XmltvID: "VRT1.be",
		SiteID:  "vrt1",
		Lang:    "nl",
	}

	programs, err := vrtbe.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Mr. Magoo" {
		t.Fatalf("expected title 'Mr. Magoo', got %q", p.Title)
	}

	if p.Description != "Rondleiding door de stad" {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	expectedStart := time.Date(2026, 3, 9, 5, 0, 8, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	if p.Channel != "VRT1.be" {
		t.Fatalf("expected channel 'VRT1.be', got %q", p.Channel)
	}

	if p.Source != "vrt.be" {
		t.Fatalf("expected source 'vrt.be', got %q", p.Source)
	}

	if p.Image == "" {
		t.Fatal("expected non-empty image")
	}
}

func TestParseEmpty(t *testing.T) {
	data := []byte(`{"data":{"page":{"previous":null,"next":null}}}`)
	programs, err := vrtbe.Parse(data, domain.Channel{XmltvID: "test"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := vrtbe.Parse([]byte("not json"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse should not error, got: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
