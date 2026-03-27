package ionplustv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/ionplustv"
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

	date := time.Date(2022, 11, 8, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "ionplustv-ionplus",
		XmltvID: "IONPlus.us",
		SiteID:  "ionplus",
		Lang:    "en",
	}

	programs, err := ionplustv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	expectedStart := time.Date(2022, 11, 8, 10, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 11, 8, 11, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Title != "All For Nothing?" {
		t.Fatalf("expected title 'All For Nothing?', got %q", p.Title)
	}

	if p.Channel != "IONPlus.us" {
		t.Fatalf("expected channel 'IONPlus.us', got %q", p.Channel)
	}

	if p.Source != "ionplustv.com" {
		t.Fatalf("expected source 'ionplustv.com', got %q", p.Source)
	}

	if p.Image == "" {
		t.Fatal("expected image to be set")
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no-content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := ionplustv.Parse(data, domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := ionplustv.Parse([]byte(""), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
