package awilime_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/awilime"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		XmltvID: "BudapestEuropaTelevizio.hu",
		SiteID:  "budapest_europa_tv",
		Lang:    "hu",
	}

	date := time.Date(2024, 6, 26, 0, 0, 0, 0, time.UTC)
	programs, err := awilime.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 15 {
		t.Fatalf("expected 15 programs, got %d", len(programs))
	}

	p := programs[3]
	expectedStart := time.Date(2024, 6, 26, 7, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2024, 6, 26, 8, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	channel := domain.Channel{SiteID: "budapest_europa_tv"}
	date := time.Date(2024, 6, 26, 0, 0, 0, 0, time.UTC)

	programs, err := awilime.Parse([]byte(`<html><head><title>Object moved</title></head><body><h2>Object moved to <a href="/tv/napi_musor/budapest_europa_tv/2024_06_24">here</a>.</h2></body></html>`), channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
