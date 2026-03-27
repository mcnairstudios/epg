package musortv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/musortv"
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
		ID:      "musor.tv-ID",
		XmltvID: "InvestigationDiscovery.hu@SD",
		SiteID:  "ID",
		Lang:    "hu",
	}

	programs, err := musortv.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	expectedStart := time.Date(2025, 10, 10, 23, 5, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 10, 11, 0, 50, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Source != "musor.tv" {
		t.Fatalf("expected source 'musor.tv', got %q", p.Source)
	}

	if p.Channel != "InvestigationDiscovery.hu@SD" {
		t.Fatalf("expected channel 'InvestigationDiscovery.hu@SD', got %q", p.Channel)
	}

	if p.Lang != "hu" {
		t.Fatalf("expected lang 'hu', got %q", p.Lang)
	}

	if p.Title == "" {
		t.Fatal("expected title to be set")
	}
}

func TestParseNewTimeFormat(t *testing.T) {
	html := `<html><body>
<div class="progarea"><div class="progentry"><div class="progentry_internal">
<time class="progentry_time" itemprop="startDate" content="2026-03-27T20:00:00Z"> 20:00</time>
<h3><a href="/test">Test Program</a></h3>
</div></div><div class="progentry"><div class="progentry_internal">
<time class="progentry_time" itemprop="startDate" content="2026-03-27T21:30:00Z"> 21:30</time>
<h3><a href="/test2">Second Program</a></h3>
</div></div></div>
</body></html>`

	ch := domain.Channel{
		ID:      "musor.tv-RTL",
		XmltvID: "RTL.hu@SD",
		SiteID:  "RTL",
		Lang:    "hu",
	}

	programs, err := musortv.Parse([]byte(html), ch)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	expectedStart := time.Date(2026, 3, 27, 20, 0, 0, 0, time.UTC)
	if !programs[0].Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, programs[0].Start)
	}

	expectedStop := time.Date(2026, 3, 27, 21, 30, 0, 0, time.UTC)
	if !programs[0].Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, programs[0].Stop)
	}

	if programs[0].Title != "Test Program" {
		t.Fatalf("expected title 'Test Program', got %q", programs[0].Title)
	}

	if programs[0].Channel != "RTL.hu@SD" {
		t.Fatalf("expected channel 'RTL.hu@SD', got %q", programs[0].Channel)
	}

	if programs[0].Lang != "hu" {
		t.Fatalf("expected lang 'hu', got %q", programs[0].Lang)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := musortv.Parse(
		[]byte("<!DOCTYPE html><html><head></head><body></body></html>"),
		domain.Channel{SiteID: "ID", Lang: "hu"},
	)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
