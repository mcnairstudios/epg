package tvmusorhu_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvmusorhu"
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
		ID:      "tvmusor-m4",
		XmltvID: "M4Sport.hu",
		SiteID:  "290",
		Lang:    "hu",
	}

	date := time.Date(2022, 11, 19, 0, 0, 0, 0, time.UTC)

	programs, err := tvmusorhu.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title == "" {
		t.Fatal("expected non-empty title")
	}

	if p.Source != "tvmusor.hu" {
		t.Fatalf("expected source 'tvmusor.hu', got %q", p.Source)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := tvmusorhu.Parse([]byte(`not json`), domain.Channel{}, time.Now())
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseMissingBlock(t *testing.T) {
	programs, err := tvmusorhu.Parse(
		[]byte(`{"data":{"loadedBlocks":{}}}`),
		domain.Channel{SiteID: "999"},
		time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
