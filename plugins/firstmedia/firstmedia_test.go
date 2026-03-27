package firstmedia_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/firstmedia"
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

	date := time.Date(2023, 11, 8, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "firstmedia-243",
		XmltvID: "AlJazeeraEnglish.qa",
		SiteID:  "243",
		Lang:    "en",
	}

	programs, err := firstmedia.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "News Live" {
		t.Fatalf("expected title 'News Live', got %q", p.Title)
	}
	if p.Description != "Up to date news and analysis from around the world." {
		t.Fatalf("expected description about news, got %q", p.Description)
	}

	expectedStart := time.Date(2023, 11, 7, 20, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
	expectedStop := time.Date(2023, 11, 7, 20, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := firstmedia.Parse([]byte(``), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := firstmedia.Parse([]byte(`not json`), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatal("expected no error")
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
