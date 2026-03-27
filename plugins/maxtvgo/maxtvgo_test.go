package maxtvgo_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/maxtvgo"
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
		ID:      "maxtvgo-105",
		XmltvID: "MaxTV.mk",
		SiteID:  "105",
		Lang:    "mk",
	}

	programs, err := maxtvgo.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	if programs[0].Source != "maxtvgo.mk" {
		t.Fatalf("expected source 'maxtvgo.mk', got %q", programs[0].Source)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := maxtvgo.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatalf("expected no error on invalid JSON, got %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNoDescription(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_no_description.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "maxtvgo-105",
		XmltvID: "MaxTV.mk",
		SiteID:  "105",
		Lang:    "mk",
	}

	programs, err := maxtvgo.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	if programs[0].Description != "" {
		t.Fatalf("expected empty description, got %q", programs[0].Description)
	}

	if programs[0].Image == "" {
		t.Fatal("expected image to be set")
	}
}

func TestParseEmpty(t *testing.T) {
	programs, err := maxtvgo.Parse([]byte(`{"programme":[]}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
