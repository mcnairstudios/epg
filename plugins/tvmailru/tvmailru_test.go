package tvmailru_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/plugins/tvmailru"
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

	date := time.Date(2021, 11, 24, 0, 0, 0, 0, time.UTC)
	programs, err := tvmailru.Parse(data, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 4 {
		t.Fatalf("expected 4 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Start.Format("2006-01-02T15:04:05.000Z") != "2021-11-24T20:35:00.000Z" {
		t.Fatalf("unexpected start: %v", p.Start)
	}
	if p.Stop.Format("2006-01-02T15:04:05.000Z") != "2021-11-24T22:40:00.000Z" {
		t.Fatalf("unexpected stop: %v", p.Stop)
	}
	if p.Category != "Музыка" {
		t.Fatalf("expected category 'Музыка', got %q", p.Category)
	}

	p2 := programs[1]
	if p2.Category != "Сериал" {
		t.Fatalf("expected category 'Сериал', got %q", p2.Category)
	}

	p3 := programs[2]
	if p3.Start.Format("2006-01-02T15:04:05.000Z") != "2021-11-24T23:40:00.000Z" {
		t.Fatalf("unexpected start for program 3: %v", p3.Start)
	}
	if p3.Category != "Позновательное" {
		t.Fatalf("expected category 'Позновательное', got %q", p3.Category)
	}

	p4 := programs[3]
	if p4.Start.Format("2006-01-02T15:04:05.000Z") != "2021-11-25T00:25:00.000Z" {
		t.Fatalf("unexpected start for program 4: %v", p4.Start)
	}
	if p4.Category != "Музыка" {
		t.Fatalf("expected category 'Музыка', got %q", p4.Category)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2021, 11, 24, 0, 0, 0, 0, time.UTC)
	programs, err := tvmailru.Parse(data, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	date := time.Date(2021, 11, 24, 0, 0, 0, 0, time.UTC)
	programs, err := tvmailru.Parse([]byte(`not json`), date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
