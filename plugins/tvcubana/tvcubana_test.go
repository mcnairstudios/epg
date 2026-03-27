package tvcubana_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvcubana"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

var doubleEscapedUnicode = regexp.MustCompile(`\\\\u([0-9a-fA-F]{4})`)

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	content := doubleEscapedUnicode.ReplaceAllString(string(data), `\u$1`)

	channel := domain.Channel{
		XmltvID: "Cubavision.cu",
		SiteID:  "cv",
		Lang:    "es",
	}

	programs, err := tvcubana.Parse([]byte(content), channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "CARIBE NOTICIAS" {
		t.Fatalf("expected title 'CARIBE NOTICIAS', got %q", p.Title)
	}

	expectedStart := time.Date(2021, 11, 22, 5, 40, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2021, 11, 22, 5, 50, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Description != "EMISIÓN DE CIERRE." {
		t.Fatalf("expected description 'EMISIÓN DE CIERRE.', got %q", p.Description)
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{SiteID: "cv"}
	programs, err := tvcubana.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := tvcubana.Parse([]byte(""), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
