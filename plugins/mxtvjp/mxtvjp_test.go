package mxtvjp_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/mxtvjp"
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
		ID:      "mxtvjp-2",
		XmltvID: "TokyoMX2.jp",
		SiteID:  "2",
		Lang:    "ja",
	}

	programs, err := mxtvjp.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "ヒーリングタイム＆ヘッドラインニュース" {
		t.Fatalf("expected title 'ヒーリングタイム＆ヘッドラインニュース', got %q", p.Title)
	}

	if p.Description != "ねこの足跡" {
		t.Fatalf("expected description 'ねこの足跡', got %q", p.Description)
	}

	expectedStart := time.Date(2024, 7, 26, 20, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2024, 7, 26, 21, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "TokyoMX2.jp" {
		t.Fatalf("expected channel 'TokyoMX2.jp', got %q", p.Channel)
	}

	if p.Source != "s.mxtv.jp" {
		t.Fatalf("expected source 's.mxtv.jp', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := mxtvjp.Parse([]byte(`[]`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := mxtvjp.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
