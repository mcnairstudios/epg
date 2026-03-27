package mtelba_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/mtelba"
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
		ID:      "mtelba-rtrs",
		XmltvID: "RTRS.ba",
		SiteID:  "msat#ch-11-rtrs",
		Lang:    "bs",
	}

	programs, err := mtelba.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 38 {
		t.Fatalf("expected 38 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Neka pesma kaže" {
		t.Fatalf("expected title 'Neka pesma kaže', got %q", p.Title)
	}
	expectedStart := time.Date(2025, 2, 3, 22, 38, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
	expectedStop := time.Date(2025, 2, 3, 23, 38, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
	if p.Image == "" {
		t.Fatal("expected image to be set")
	}

	p37 := programs[37]
	if p37.Title != "Bitanga s plaže" {
		t.Fatalf("expected title 'Bitanga s plaže', got %q", p37.Title)
	}
	expectedStart37 := time.Date(2025, 2, 4, 22, 27, 0, 0, time.UTC)
	if !p37.Start.Equal(expectedStart37) {
		t.Fatalf("expected start %v, got %v", expectedStart37, p37.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := mtelba.Parse([]byte(`{}`), domain.Channel{SiteID: "msat#ch-11-rtrs"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := mtelba.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatal("expected no error")
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
