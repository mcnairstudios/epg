package ontvtonight_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/ontvtonight"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseResponse(t *testing.T) {
	content, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "au#1692/7two",
		XmltvID: "7two.au",
	}

	date := time.Date(2021, 11, 25, 0, 0, 0, 0, time.UTC)

	programs, err := ontvtonight.Parse(content, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 3 {
		t.Fatalf("expected 3 programs, got %d", len(programs))
	}

	p0 := programs[0]
	if p0.Title != "What A Carry On" {
		t.Fatalf("expected title 'What A Carry On', got %q", p0.Title)
	}

	expectedStart0 := "2021-11-24T13:10:00Z"
	if p0.Start.UTC().Format("2006-01-02T15:04:05Z") != expectedStart0 {
		t.Fatalf("expected start %s, got %s", expectedStart0, p0.Start.UTC().Format("2006-01-02T15:04:05Z"))
	}

	expectedStop0 := "2021-11-24T13:50:00Z"
	if p0.Stop.UTC().Format("2006-01-02T15:04:05Z") != expectedStop0 {
		t.Fatalf("expected stop %s, got %s", expectedStop0, p0.Stop.UTC().Format("2006-01-02T15:04:05Z"))
	}

	p1 := programs[1]
	if p1.Title != "Bones" {
		t.Fatalf("expected title 'Bones', got %q", p1.Title)
	}

	if p1.Description != "The Devil In The Details" {
		t.Fatalf("expected description 'The Devil In The Details', got %q", p1.Description)
	}

	expectedStart1 := "2021-11-24T13:50:00Z"
	if p1.Start.UTC().Format("2006-01-02T15:04:05Z") != expectedStart1 {
		t.Fatalf("expected start %s, got %s", expectedStart1, p1.Start.UTC().Format("2006-01-02T15:04:05Z"))
	}

	expectedStop1 := "2021-11-25T11:50:00Z"
	if p1.Stop.UTC().Format("2006-01-02T15:04:05Z") != expectedStop1 {
		t.Fatalf("expected stop %s, got %s", expectedStop1, p1.Stop.UTC().Format("2006-01-02T15:04:05Z"))
	}

	p2 := programs[2]
	if p2.Title != "Inspector Morse: The Remorseful Day" {
		t.Fatalf("expected title 'Inspector Morse: The Remorseful Day', got %q", p2.Title)
	}

	expectedStart2 := "2021-11-25T11:50:00Z"
	if p2.Start.UTC().Format("2006-01-02T15:04:05Z") != expectedStart2 {
		t.Fatalf("expected start %s, got %s", expectedStart2, p2.Start.UTC().Format("2006-01-02T15:04:05Z"))
	}

	expectedStop2 := "2021-11-25T12:50:00Z"
	if p2.Stop.UTC().Format("2006-01-02T15:04:05Z") != expectedStop2 {
		t.Fatalf("expected stop %s, got %s", expectedStop2, p2.Stop.UTC().Format("2006-01-02T15:04:05Z"))
	}
}

func TestParseEmptyContent(t *testing.T) {
	content, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "au#1692/7two",
		XmltvID: "7two.au",
	}

	date := time.Date(2021, 11, 25, 0, 0, 0, 0, time.UTC)

	programs, err := ontvtonight.Parse(content, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
