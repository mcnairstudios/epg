package onetpl_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/onetpl"
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

	date := time.Date(2021, 11, 24, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "onetpl-13thstreet",
		XmltvID: "13thStreet.de",
		SiteID:  "13th-street-250",
		Lang:    "pl",
	}

	programs, err := onetpl.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 3 {
		t.Fatalf("expected 3 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Law & Order, odc. 15: Letzte Worte" {
		t.Fatalf("expected title about Law & Order, got %q", p.Title)
	}
	if p.Category != "Krimiserie" {
		t.Fatalf("expected category 'Krimiserie', got %q", p.Category)
	}
	expectedStart := time.Date(2021, 11, 24, 2, 20, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
	expectedStop := time.Date(2021, 11, 24, 22, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	p1 := programs[1]
	if p1.Title != "Navy CIS, odc. 1: New Orleans" {
		t.Fatalf("expected title about Navy CIS, got %q", p1.Title)
	}
	expectedStart1 := time.Date(2021, 11, 24, 22, 30, 0, 0, time.UTC)
	if !p1.Start.Equal(expectedStart1) {
		t.Fatalf("expected start %v, got %v", expectedStart1, p1.Start)
	}
	expectedStop1 := time.Date(2021, 11, 25, 0, 0, 0, 0, time.UTC)
	if !p1.Stop.Equal(expectedStop1) {
		t.Fatalf("expected stop %v, got %v", expectedStop1, p1.Stop)
	}

	p2 := programs[2]
	if p2.Title != "Navy CIS: L.A, odc. 13: High Society" {
		t.Fatalf("expected title about Navy CIS LA, got %q", p2.Title)
	}
	expectedStart2 := time.Date(2021, 11, 25, 0, 0, 0, 0, time.UTC)
	if !p2.Start.Equal(expectedStart2) {
		t.Fatalf("expected start %v, got %v", expectedStart2, p2.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := onetpl.Parse(data, domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
