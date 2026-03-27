package winplay_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/winplay"
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
		ID:      "winplay-futbol",
		XmltvID: "WinFutbol.co",
		SiteID:  "529cff6f6bd2ea6b610000e0",
		Lang:    "es",
	}

	programs, err := winplay.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	if programs[0].Source != "winplay.co" {
		t.Fatalf("expected source 'winplay.co', got %q", programs[0].Source)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := winplay.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseEmpty(t *testing.T) {
	programs, err := winplay.Parse([]byte(`{"data":{"getLives":[]}}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
