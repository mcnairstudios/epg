package tvlv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvlv"
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
		SiteID:  "ltv1",
		XmltvID: "LTV1.lv",
		Lang:    "lv",
	}

	programs, err := tvlv.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Ziņas" {
		t.Fatalf("expected title %q, got %q", "Ziņas", p.Title)
	}
	if p.Description != "Latvijas ziņu izlaidums" {
		t.Fatalf("expected description %q, got %q", "Latvijas ziņu izlaidums", p.Description)
	}
	if p.Category != "News" {
		t.Fatalf("expected category %q, got %q", "News", p.Category)
	}
	if p.Image != "https://www.tv.lv/images/zinas.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	expectedStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := tvlv.Parse([]byte("not json"), domain.Channel{})
	if err != nil {
		t.Fatalf("expected nil error for invalid JSON, got: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptySchedule(t *testing.T) {
	data := []byte(`{"schedule":{"programme":[]}}`)
	programs, err := tvlv.Parse(data, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
