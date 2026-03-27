package porthu_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/porthu"
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
		ID:      "porthu-5",
		XmltvID: "TV2.hu",
		SiteID:  "5",
		Lang:    "hu",
	}

	programs, err := porthu.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 25 {
		t.Fatalf("expected 25 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Terhes társaság" {
		t.Fatalf("expected title 'Terhes társaság', got %q", p.Title)
	}
	if p.Description != "amerikai vígjáték, 2010" {
		t.Fatalf("expected description 'amerikai vígjáték, 2010', got %q", p.Description)
	}
	if p.Category != "film" {
		t.Fatalf("expected category 'film', got %q", p.Category)
	}
	if p.Start.Format("2006-01-02T15:04:05.000Z") != "2025-11-09T21:40:00.000Z" {
		t.Fatalf("unexpected start: %v", p.Start)
	}
	if p.Stop.Format("2006-01-02T15:04:05.000Z") != "2025-11-09T23:45:00.000Z" {
		t.Fatalf("unexpected stop: %v", p.Stop)
	}

	p24 := programs[24]
	if p24.Title != "CSI: A helyszínelők" {
		t.Fatalf("expected title 'CSI: A helyszínelők', got %q", p24.Title)
	}
	if p24.SubTitle != "Bukott bálványok" {
		t.Fatalf("expected subtitle 'Bukott bálványok', got %q", p24.SubTitle)
	}
	if p24.Description != "amerikai-kanadai krimisorozat, VII / 17. rész, 2007" {
		t.Fatalf("expected description about krimisorozat, got %q", p24.Description)
	}
	if p24.Category != "filmsorozat" {
		t.Fatalf("expected category 'filmsorozat', got %q", p24.Category)
	}
	if p24.Start.Format("2006-01-02T15:04:05.000Z") != "2025-11-11T01:50:00.000Z" {
		t.Fatalf("unexpected start: %v", p24.Start)
	}
	if p24.Stop.Format("2006-01-02T15:04:05.000Z") != "2025-11-11T03:00:00.000Z" {
		t.Fatalf("unexpected stop: %v", p24.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{SiteID: "5"}
	programs, err := porthu.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := porthu.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatal("expected no error on invalid JSON")
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
