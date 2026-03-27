package novacyprus_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/novacyprus"
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
		ID:      "novacyprus-1",
		XmltvID: "NovaCinema1.cy",
		SiteID:  "614",
		Lang:    "el",
	}

	programs, err := novacyprus.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	if programs[0].Source != "novacyprus.com" {
		t.Fatalf("expected source 'novacyprus.com', got %q", programs[0].Source)
	}

	if programs[0].Image != "http://cache-forthnet.secure.footprint.net/linear/3/0/305608_COMMOBLOOX_GUIDE_STILL.jpg" {
		t.Fatalf("unexpected image: %q", programs[0].Image)
	}

	if programs[0].Title != "Δεσμοί Αίματος" {
		t.Fatalf("unexpected title: %q", programs[0].Title)
	}

	expectedStart := "2021-11-17T04:20:00.000Z"
	if programs[0].Start.Format("2006-01-02T15:04:05.000Z") != expectedStart {
		t.Fatalf("expected start %s, got %v", expectedStart, programs[0].Start)
	}

	expectedStop := "2021-11-17T06:10:00.000Z"
	if programs[0].Stop.Format("2006-01-02T15:04:05.000Z") != expectedStop {
		t.Fatalf("expected stop %s, got %v", expectedStop, programs[0].Stop)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := novacyprus.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseEmpty(t *testing.T) {
	programs, err := novacyprus.Parse([]byte(`{"nodes":[]}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
