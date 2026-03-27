package mojiid_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/mojiid"
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

	channel := domain.Channel{
		ID:      "moji.id-moji",
		XmltvID: "MOJI.id",
		SiteID:  "moji",
		Lang:    "id",
	}

	date := time.Date(2023, 8, 18, 0, 0, 0, 0, time.UTC)
	programs, err := mojiid.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "TRUST" {
		t.Fatalf("expected title 'TRUST', got %q", p.Title)
	}

	if p.Description != "Informasi seputar menjaga vitalitas pria" {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	expectedStart := time.Date(2023, 8, 17, 17, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 8, 17, 17, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Source != "moji.id" {
		t.Fatalf("expected source 'moji.id', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := mojiid.Parse([]byte(""), domain.Channel{}, time.Time{})
	if err != nil {
		t.Fatalf("parse should not fail: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
