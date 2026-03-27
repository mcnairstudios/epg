package mediaklikk_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/plugins/mediaklikk"
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

	programs, err := mediaklikk.Parse(data)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "A hegyi doktor - I. évad" {
		t.Fatalf("expected title 'A hegyi doktor - I. évad', got %q", p.Title)
	}
	if p.Start.Format("2006-01-02T15:04:05.000Z") != "2022-10-27T22:00:46.000Z" {
		t.Fatalf("unexpected start: %v", p.Start)
	}
	if p.Stop.Format("2006-01-02T15:04:05.000Z") != "2022-10-27T22:54:00.000Z" {
		t.Fatalf("unexpected stop: %v", p.Stop)
	}
	if p.Image != "https://mediaklikk.hu/wp-content/uploads/sites/4/2019/10/A-hegyi-doktor-I-évad-e1571318391226-150x150.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	if len(programs) < 57 {
		t.Fatalf("expected at least 57 programs, got %d", len(programs))
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := mediaklikk.Parse([]byte("<!DOCTYPE html><html><head></head><body></body></html>"))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidHTML(t *testing.T) {
	programs, err := mediaklikk.Parse([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
