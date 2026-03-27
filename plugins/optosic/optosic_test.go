package optosic_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/optosic"
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
		SiteID:  "38719848-2a57-42e3-8640-63a9aa39f107",
		XmltvID: "SICNoticias.pt",
		Lang:    "pt",
	}

	programs, err := optosic.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 17 {
		t.Fatalf("expected 17 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "JORNAL DA MEIA-NOITE" {
		t.Fatalf("expected title 'JORNAL DA MEIA-NOITE', got %q", p.Title)
	}
	if p.Episode != 16 {
		t.Fatalf("expected episode 16, got %d", p.Episode)
	}

	expectedStart := time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 1, 17, 1, 45, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	last := programs[16]
	if last.Title != "EXPRESSO DA MEIA-NOITE" {
		t.Fatalf("expected last title 'EXPRESSO DA MEIA-NOITE', got %q", last.Title)
	}
	if last.Episode != 2 {
		t.Fatalf("expected episode 2, got %d", last.Episode)
	}

	expectedStart = time.Date(2025, 1, 17, 23, 0, 0, 0, time.UTC)
	if !last.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, last.Start)
	}

	expectedStop = time.Date(2025, 1, 18, 0, 0, 0, 0, time.UTC)
	if !last.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, last.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := optosic.Parse([]byte(""), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := optosic.Parse([]byte("not json"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
