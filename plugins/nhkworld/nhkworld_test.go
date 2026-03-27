package nhkworld_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/nhkworld"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContentEN(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_en.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "nhkworld-en",
		XmltvID: "NHKWorldPremium.jp",
		SiteID:  "#",
		Lang:    "en",
	}

	date := time.Date(2023, 7, 10, 0, 0, 0, 0, time.UTC)

	programs, err := nhkworld.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 56 {
		t.Fatalf("expected 56 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "NHK Amateur Singing Contest" {
		t.Fatalf("unexpected title: %q", p.Title)
	}

	expectedStart := time.Date(2023, 7, 9, 15, 35, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 7, 9, 16, 20, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseContentJA(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_ja.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "nhkworld-ja",
		XmltvID: "NHKWorldPremium.jp",
		SiteID:  "#",
		Lang:    "ja",
	}

	date := time.Date(2023, 7, 10, 0, 0, 0, 0, time.UTC)

	programs, err := nhkworld.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 56 {
		t.Fatalf("expected 56 programs, got %d", len(programs))
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := nhkworld.Parse([]byte(`{}`), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := nhkworld.Parse([]byte(`not json`), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse should not error, got: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
