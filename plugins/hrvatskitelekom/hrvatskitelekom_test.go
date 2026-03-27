package hrvatskitelekom_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/hrvatskitelekom"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseMultiContent(t *testing.T) {
	data0, err := os.ReadFile(testdataPath("content_0.json"))
	if err != nil {
		t.Fatalf("failed to read content_0.json: %v", err)
	}
	data3, err := os.ReadFile(testdataPath("content_3.json"))
	if err != nil {
		t.Fatalf("failed to read content_3.json: %v", err)
	}
	data21, err := os.ReadFile(testdataPath("content_21.json"))
	if err != nil {
		t.Fatalf("failed to read content_21.json: %v", err)
	}

	channel := domain.Channel{
		ID:      "mojmaxtv.hrvatskitelekom.hr-274913832105",
		XmltvID: "HRT1.hr",
		SiteID:  "274913832105",
		Lang:    "hr",
	}

	programs, err := hrvatskitelekom.ParseMulti([][]byte{data0, data3, data21}, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 17 {
		t.Fatalf("expected 17 programs, got %d", len(programs))
	}

	first := programs[0]
	if first.Title != "Planet Zemlja: Junaci" {
		t.Fatalf("expected title 'Planet Zemlja: Junaci', got %q", first.Title)
	}

	if first.Category != "Dokumentarni" {
		t.Fatalf("expected category 'Dokumentarni', got %q", first.Category)
	}

	if first.Season != 3 {
		t.Fatalf("expected season 3, got %d", first.Season)
	}

	if first.Episode != 8 {
		t.Fatalf("expected episode 8, got %d", first.Episode)
	}

	expectedStart := time.Date(2025, 1, 23, 23, 16, 0, 0, time.UTC)
	if !first.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, first.Start)
	}

	expectedStop := time.Date(2025, 1, 24, 0, 8, 0, 0, time.UTC)
	if !first.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, first.Stop)
	}

	last := programs[len(programs)-1]
	if last.Title != "Harry Haft, film" {
		t.Fatalf("expected last title 'Harry Haft, film', got %q", last.Title)
	}

	lastStart := time.Date(2025, 1, 24, 21, 50, 0, 0, time.UTC)
	if !last.Start.Equal(lastStart) {
		t.Fatalf("expected last start %v, got %v", lastStart, last.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := hrvatskitelekom.Parse([]byte(`{}`), domain.Channel{SiteID: "274913832105"})
	if err != nil {
		t.Fatalf("parse should not fail: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := hrvatskitelekom.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse should not fail: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
