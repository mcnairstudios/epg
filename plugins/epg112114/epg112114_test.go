package epg112114_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/epg112114"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.xml"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2025, 1, 11, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "epg112114-btv-wenyi",
		XmltvID: "BRTVArtsChannel.cn",
		SiteID:  "BTV文艺",
		Lang:    "zh",
	}

	programs, err := epg112114.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 28 {
		t.Fatalf("expected 28 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "每日文艺播报" {
		t.Fatalf("expected title '每日文艺播报', got %q", p.Title)
	}
	if p.Start.Format("2006-01-02T15:04:05.000Z") != "2025-01-11T00:07:00.000Z" {
		t.Fatalf("unexpected start: %v", p.Start)
	}
	if p.Stop.Format("2006-01-02T15:04:05.000Z") != "2025-01-11T00:24:00.000Z" {
		t.Fatalf("unexpected stop: %v", p.Stop)
	}

	last := programs[27]
	if last.Title != "笑动剧场" {
		t.Fatalf("expected title '笑动剧场', got %q", last.Title)
	}
	if last.Start.Format("2006-01-02T15:04:05.000Z") != "2025-01-11T15:16:00.000Z" {
		t.Fatalf("unexpected start for last: %v", last.Start)
	}
	if last.Stop.Format("2006-01-02T15:04:05.000Z") != "2025-01-11T15:59:00.000Z" {
		t.Fatalf("unexpected stop for last: %v", last.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	date := time.Date(2025, 1, 11, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{SiteID: "BTV文艺", Lang: "zh"}
	programs, err := epg112114.Parse([]byte(""), channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidXML(t *testing.T) {
	date := time.Date(2025, 1, 11, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{SiteID: "BTV文艺", Lang: "zh"}
	programs, err := epg112114.Parse([]byte("not xml"), channel, date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
