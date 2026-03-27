package clarobr2_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/clarobr2"
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
		SiteID:  "1_1682",
		XmltvID: "TVCamara.br",
		Lang:    "pt",
	}

	programs, err := clarobr2.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 23 {
		t.Fatalf("expected 23 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Encontro Com a Autora" {
		t.Fatalf("expected title 'Encontro Com a Autora', got %q", p.Title)
	}

	expectedStart := time.Date(2025, 8, 13, 3, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 8, 13, 4, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	last := programs[22]
	if last.Title != "Palavra Aberta" {
		t.Fatalf("expected title 'Palavra Aberta', got %q", last.Title)
	}

	expectedStart = time.Date(2025, 8, 14, 2, 30, 0, 0, time.UTC)
	if !last.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, last.Start)
	}

	expectedStop = time.Date(2025, 8, 14, 3, 0, 0, 0, time.UTC)
	if !last.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, last.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := clarobr2.Parse(data, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := clarobr2.Parse([]byte("not json"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
