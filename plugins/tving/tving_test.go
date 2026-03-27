package tving_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tving"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.txt"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "C00551",
		XmltvID: "tvN.kr",
		Lang:    "ko",
	}

	programs, err := tving.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "외계+인 1부" {
		t.Fatalf("expected title '외계+인 1부', got %q", p.Title)
	}

	expectedStart := time.Date(2023, 1, 22, 13, 40, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 1, 22, 15, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image != "https://image.tving.com/upload/cms/caip/CAIP0200/P001661154.jpg" {
		t.Fatalf("expected image with tving.com prefix, got %q", p.Image)
	}

	if p.Channel != "tvN.kr" {
		t.Fatalf("expected channel 'tvN.kr', got %q", p.Channel)
	}

	if p.Source != "m.tving.com" {
		t.Fatalf("expected source 'm.tving.com', got %q", p.Source)
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.txt"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := tving.Parse(data, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := tving.Parse([]byte{}, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
