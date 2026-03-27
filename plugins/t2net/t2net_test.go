package t2net_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/t2net"
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
		ID:      "tv2go.t-2.net-1000259",
		XmltvID: "TVSlovenija1.si",
		SiteID:  "1000259",
		Lang:    "sl",
	}

	programs, err := t2net.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Dnevnik Slovencev v Italiji" {
		t.Fatalf("expected title 'Dnevnik Slovencev v Italiji', got %q", p.Title)
	}

	if p.Category != "Informativni" {
		t.Fatalf("expected category 'Informativni', got %q", p.Category)
	}

	if p.Description != "Dnevnik Slovencev v Italiji je informativna oddaja, v kateri novinarji poro\u010dajo predvsem o dnevnih dogodkih med Slovenci v Italiji." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	if p.Image != "https://tv2go.t-2.net/static/media/img/epg/max_crop/EPG_IMG_2927405.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	expectedStart := time.Date(2021, 11, 19, 0, 50, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2021, 11, 19, 1, 15, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := t2net.Parse([]byte(`{}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse should not fail: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidContent(t *testing.T) {
	programs, err := t2net.Parse([]byte("Invalid API client identifier"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse should not fail: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
