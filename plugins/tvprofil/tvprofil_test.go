package tvprofil_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvprofil"
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
		ID:      "tvprofil.com-bg/tv-programa#24kitchen-bg",
		XmltvID: "24KitchenBulgaria.bg",
		SiteID:  "bg/tv-programa#24kitchen-bg",
		Lang:    "bg",
	}

	programs, err := tvprofil.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Save with Jamie 1, ep. 2" {
		t.Fatalf("expected title 'Save with Jamie 1, ep. 2', got %q", p.Title)
	}

	expectedStart := time.Date(2025, 7, 29, 5, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 7, 29, 6, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Source != "tvprofil.com" {
		t.Fatalf("expected source 'tvprofil.com', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.txt"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := tvprofil.Parse(data, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNoData(t *testing.T) {
	programs, err := tvprofil.Parse([]byte(""), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
