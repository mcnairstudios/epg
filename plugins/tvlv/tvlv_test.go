package tvlv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvlv"
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
		ID:      "tvlv-ltv1",
		XmltvID: "LTV1.lv",
		SiteID:  "ltv1",
		Lang:    "lv",
	}

	programs, err := tvlv.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 40 {
		t.Fatalf("expected 40 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Ielas garumā.  Pārdaugavas koka arhitektūra" {
		t.Fatalf("unexpected title: %q", p.Title)
	}

	expectedStart := time.Date(2023, 11, 29, 22, 5, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 11, 29, 22, 35, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	last := programs[39]
	if last.Title != "Latvijas Sirdsdziesma" {
		t.Fatalf("unexpected last title: %q", last.Title)
	}

	expectedLastStart := time.Date(2023, 11, 30, 21, 30, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}

	expectedLastStop := time.Date(2023, 11, 30, 22, 30, 0, 0, time.UTC)
	if !last.Stop.Equal(expectedLastStop) {
		t.Fatalf("expected last stop %v, got %v", expectedLastStop, last.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := tvlv.Parse(data, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := tvlv.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
