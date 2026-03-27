package meopt_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/meopt"
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
		SiteID:  "RTP1",
		XmltvID: "RTP1.pt",
		Lang:    "pt",
	}

	programs, err := meopt.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Bom Dia Portugal" {
		t.Fatalf("expected title %q, got %q", "Bom Dia Portugal", p.Title)
	}
	if p.Description != "Programa matinal de informação" {
		t.Fatalf("expected description %q, got %q", "Programa matinal de informação", p.Description)
	}
	if p.Lang != "pt" {
		t.Fatalf("expected lang %q, got %q", "pt", p.Lang)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := meopt.Parse([]byte("not json"), domain.Channel{})
	if err != nil {
		t.Fatalf("expected nil error for invalid JSON, got: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyData(t *testing.T) {
	programs, err := meopt.Parse([]byte(""), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseErrorStatus(t *testing.T) {
	data := []byte(`{"Status":"Error","StatusCode":500,"Result":[]}`)
	programs, err := meopt.Parse(data, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNoTitle(t *testing.T) {
	data := []byte(`{"Status":"OK","StatusCode":200,"Result":[{"CallLetter":"RTP1","StartDate":"2024-01-01T10:00:00","EndDate":"2024-01-01T11:00:00","Title":"","Synopsis":""}]}`)
	channel := domain.Channel{SiteID: "RTP1", XmltvID: "RTP1.pt", Lang: "pt"}
	programs, err := meopt.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}
	if programs[0].Title != "Sem título" {
		t.Fatalf("expected fallback title %q, got %q", "Sem título", programs[0].Title)
	}
}
