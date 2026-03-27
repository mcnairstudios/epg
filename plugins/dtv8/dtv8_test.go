package dtv8_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/dtv8"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

var channel = domain.Channel{
	ID:      "dtv8.net-1",
	XmltvID: "DTV8.gy",
	SiteID:  "1",
	Lang:    "en",
}

func TestParseFriday(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_fri.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2025, 2, 21, 0, 0, 0, 0, time.UTC)
	programs, err := dtv8.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 18 {
		t.Fatalf("expected 18 programs, got %d", len(programs))
	}

	p9 := programs[9]
	if p9.Title != "Smallville" {
		t.Fatalf("expected title 'Smallville', got %q", p9.Title)
	}
	if p9.Image != "http://dtv8.net/wp-content/uploads/71P0aShCBXL._SL1300_.jpg" {
		t.Fatalf("unexpected image: %q", p9.Image)
	}
	expectedStart := time.Date(2025, 2, 21, 21, 0, 0, 0, time.UTC)
	if !p9.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p9.Start)
	}
	expectedStop := time.Date(2025, 2, 21, 22, 0, 0, 0, time.UTC)
	if !p9.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p9.Stop)
	}

	p15 := programs[15]
	if p15.Title != "Law & Order" {
		t.Fatalf("expected title 'Law & Order', got %q", p15.Title)
	}
	expectedStart15 := time.Date(2025, 2, 22, 1, 45, 0, 0, time.UTC)
	if !p15.Start.Equal(expectedStart15) {
		t.Fatalf("expected start %v, got %v", expectedStart15, p15.Start)
	}
	expectedStop15 := time.Date(2025, 2, 22, 2, 30, 0, 0, time.UTC)
	if !p15.Stop.Equal(expectedStop15) {
		t.Fatalf("expected stop %v, got %v", expectedStop15, p15.Stop)
	}
}

func TestParseSaturday(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_sat.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2025, 2, 22, 0, 0, 0, 0, time.UTC)
	programs, err := dtv8.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 11 {
		t.Fatalf("expected 11 programs, got %d", len(programs))
	}

	p0 := programs[0]
	if p0.Title != "Sign On" {
		t.Fatalf("expected title 'Sign On', got %q", p0.Title)
	}
	expectedStart := time.Date(2025, 2, 22, 13, 55, 0, 0, time.UTC)
	if !p0.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p0.Start)
	}

	p10 := programs[10]
	if p10.Title != "Sign Off" {
		t.Fatalf("expected title 'Sign Off', got %q", p10.Title)
	}
	expectedStart10 := time.Date(2025, 2, 23, 4, 0, 0, 0, time.UTC)
	if !p10.Start.Equal(expectedStart10) {
		t.Fatalf("expected start %v, got %v", expectedStart10, p10.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := dtv8.Parse([]byte{}, channel, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
