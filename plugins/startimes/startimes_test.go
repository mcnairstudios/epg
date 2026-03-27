package startimes_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/startimes"
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

	date := time.Date(2024, 12, 10, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "startimes-1023102509",
		XmltvID: "ZeeOneAfrica.za",
		SiteID:  "1023102509",
		Lang:    "en",
	}

	programs, err := startimes.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 22 {
		t.Fatalf("expected 22 programs, got %d", len(programs))
	}

	p := programs[0]

	expectedStart := time.Date(2024, 12, 10, 0, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2024, 12, 10, 1, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Title != "Deserted S1 E37" {
		t.Fatalf("expected title 'Deserted S1 E37', got %q", p.Title)
	}

	if p.Season != 1 {
		t.Fatalf("expected season 1, got %d", p.Season)
	}

	if p.Episode != 37 {
		t.Fatalf("expected episode 37, got %d", p.Episode)
	}

	if p.Channel != "ZeeOneAfrica.za" {
		t.Fatalf("expected channel 'ZeeOneAfrica.za', got %q", p.Channel)
	}

	if p.Source != "startimestv.com" {
		t.Fatalf("expected source 'startimestv.com', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := startimes.Parse([]byte(""), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNoPrograms(t *testing.T) {
	content := `<!DOCTYPE html><html><body><div class="box"></div></body></html>`
	programs, err := startimes.Parse([]byte(content), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
