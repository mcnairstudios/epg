package programetvro_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/programetvro"
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
		ID:      "programetvro-pro-tv",
		XmltvID: "ProTV.ro",
		SiteID:  "pro-tv",
		Lang:    "ro",
	}

	programs, err := programetvro.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Ştirile Pro Tv" {
		t.Fatalf("expected title 'Ştirile Pro Tv', got %q", p.Title)
	}

	if p.Description != "În fiecare zi, cele mai importante evenimente, transmisiuni LIVE, analize, anchete şi reportaje sunt la Ştirile ProTV." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	expectedStart := time.Date(2021, 11, 7, 5, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2021, 11, 7, 7, 59, 59, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Category != "Ştiri" {
		t.Fatalf("unexpected category: %q", p.Category)
	}

	if p.Image != "https://www.programetv.ro/img/shows/84/54/stirile-pro-tv.png?key=Z2lfZnVial90cmFyZXZwLzAwLzAwLzA1LzE4MzgxMnktMTIwazE3MC1hLW40NTk4MW9zLmNhdA==" {
		t.Fatalf("unexpected image: %q", p.Image)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := programetvro.Parse(data, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidContent(t *testing.T) {
	programs, err := programetvro.Parse([]byte(`no pageData here`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
