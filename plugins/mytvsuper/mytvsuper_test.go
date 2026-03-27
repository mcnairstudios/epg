package mytvsuper_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/mytvsuper"
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

	date := time.Date(2022, 11, 15, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "mytvsuper-B",
		XmltvID: "J2.hk",
		SiteID:  "B",
		Lang:    "zh",
	}

	programs, err := mytvsuper.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Start.Format("2006-01-02T15:04:05.000Z") != "2022-11-14T22:00:00.000Z" {
		t.Fatalf("unexpected start: %v", p.Start)
	}
	if p.Stop.Format("2006-01-02T15:04:05.000Z") != "2022-11-14T23:00:00.000Z" {
		t.Fatalf("unexpected stop: %v", p.Stop)
	}
	if p.Episode != 1000003 {
		t.Fatalf("expected episode 1000003, got %d", p.Episode)
	}
}

func TestParseContentEnglish(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2022, 11, 15, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "mytvsuper-B",
		XmltvID: "J2.hk",
		SiteID:  "B",
		Lang:    "en",
	}

	programs, err := mytvsuper.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Someday or One Day#3[Can/Man][PG]" {
		t.Fatalf("expected English title, got %q", p.Title)
	}
	if p.Description != "Description" {
		t.Fatalf("expected English description, got %q", p.Description)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2022, 11, 15, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{SiteID: "B", Lang: "zh"}
	programs, err := mytvsuper.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	date := time.Date(2022, 11, 15, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{SiteID: "B", Lang: "zh"}
	programs, err := mytvsuper.Parse([]byte(`not json`), channel, date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
