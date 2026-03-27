package tvplustr_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvplustr"
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
		ID:      "tvplus.com.tr-nick-jr/4353",
		XmltvID: "NickJr.tr",
		SiteID:  "nick-jr/4353",
		Lang:    "tr",
	}

	date := time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC)
	programs, err := tvplustr.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Camgöz (2020)" {
		t.Fatalf("expected title 'Camgöz (2020)', got %q", p.Title)
	}

	expectedStart := time.Date(2024, 12, 14, 21, 10, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	if p.Season != 1 {
		t.Fatalf("expected season 1, got %d", p.Season)
	}

	if p.Episode != 116 {
		t.Fatalf("expected episode 116, got %d", p.Episode)
	}

	expectedStop := time.Date(2024, 12, 14, 21, 20, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Category != "Çocuk" {
		t.Fatalf("expected category 'Çocuk', got %q", p.Category)
	}

	if p.Channel != "NickJr.tr" {
		t.Fatalf("expected channel 'NickJr.tr', got %q", p.Channel)
	}

	if p.Source != "tvplus.com.tr" {
		t.Fatalf("expected source 'tvplus.com.tr', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := tvplustr.Parse([]byte(``), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := tvplustr.Parse([]byte(`not json`), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
