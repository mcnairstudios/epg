package tvhebdo_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvhebdo"
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
		ID:      "tvhebdo-src/CBFT",
		XmltvID: "CBFT.ca",
		SiteID:  "src/CBFT",
		Lang:    "fr",
	}

	date := time.Date(2022, 5, 11, 0, 0, 0, 0, time.UTC)
	programs, err := tvhebdo.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "5 chefs dans ma cuisine" {
		t.Fatalf("expected title '5 chefs dans ma cuisine', got %q", p.Title)
	}
	expectedStart := time.Date(2022, 5, 11, 15, 30, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
	expectedStop := time.Date(2022, 5, 11, 16, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if len(programs) < 17 {
		t.Fatalf("expected at least 17 programs, got %d", len(programs))
	}
	p16 := programs[16]
	if p16.Title != "Outlander: Le chardon et le tartan" {
		t.Fatalf("expected title 'Outlander: Le chardon et le tartan', got %q", p16.Title)
	}
	expectedStart16 := time.Date(2022, 5, 12, 4, 9, 0, 0, time.UTC)
	if !p16.Start.Equal(expectedStart16) {
		t.Fatalf("expected start %v, got %v", expectedStart16, p16.Start)
	}
	expectedStop16 := time.Date(2022, 5, 12, 5, 19, 0, 0, time.UTC)
	if !p16.Stop.Equal(expectedStop16) {
		t.Fatalf("expected stop %v, got %v", expectedStop16, p16.Stop)
	}

	if len(programs) < 37 {
		t.Fatalf("expected at least 37 programs, got %d", len(programs))
	}
	p36 := programs[36]
	if p36.Title != "Ricardo" {
		t.Fatalf("expected title 'Ricardo', got %q", p36.Title)
	}
	expectedStart36 := time.Date(2022, 5, 12, 15, 0, 0, 0, time.UTC)
	if !p36.Start.Equal(expectedStart36) {
		t.Fatalf("expected start %v, got %v", expectedStart36, p36.Start)
	}
	expectedStop36 := time.Date(2022, 5, 12, 15, 30, 0, 0, time.UTC)
	if !p36.Stop.Equal(expectedStop36) {
		t.Fatalf("expected stop %v, got %v", expectedStop36, p36.Stop)
	}

	if p.Channel != "CBFT.ca" {
		t.Fatalf("expected channel 'CBFT.ca', got %q", p.Channel)
	}
	if p.Source != "tvhebdo.com" {
		t.Fatalf("expected source 'tvhebdo.com', got %q", p.Source)
	}
	if p.Lang != "fr" {
		t.Fatalf("expected lang 'fr', got %q", p.Lang)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := tvhebdo.Parse([]byte{}, domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2022, 5, 11, 0, 0, 0, 0, time.UTC)
	programs, err := tvhebdo.Parse(data, domain.Channel{}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidHTML(t *testing.T) {
	programs, err := tvhebdo.Parse([]byte("not html at all"), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
