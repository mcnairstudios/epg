package sporttv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/sporttv"
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
		ID:      "sporttv-727",
		XmltvID: "SportTV1.pt",
		SiteID:  "727",
		Lang:    "pt",
	}

	date := time.Date(2024, 12, 23, 0, 0, 0, 0, time.UTC)

	programs, err := sporttv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 19 {
		t.Fatalf("expected 19 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "RESUMOS DA JORNADA 15" {
		t.Fatalf("unexpected title: %q", p.Title)
	}

	expectedStart := time.Date(2024, 12, 23, 1, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := sporttv.Parse([]byte(""), domain.Channel{SiteID: "727"}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
