package chadama_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/chadama"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Now()
	channel := domain.Channel{
		ID:      "chadama-chadatv",
		XmltvID: "ChadaTV.ma",
		SiteID:  "chadatv",
		Lang:    "fr",
	}

	programs, err := chadama.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Bloc Prime + Clips" {
		t.Fatalf("expected title 'Bloc Prime + Clips', got %q", p.Title)
	}

	if p.Description != "No description available" {
		t.Fatalf("expected description 'No description available', got %q", p.Description)
	}

	if p.Channel != "ChadaTV.ma" {
		t.Fatalf("expected channel 'ChadaTV.ma', got %q", p.Channel)
	}

	if p.Source != "chada.ma" {
		t.Fatalf("expected source 'chada.ma', got %q", p.Source)
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := chadama.Parse(data, domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := chadama.Parse([]byte(""), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
