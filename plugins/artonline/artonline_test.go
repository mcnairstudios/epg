package artonline_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/artonline"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

var channel = domain.Channel{
	ID:      "artonline.tv-#Aflam2",
	XmltvID: "ARTAflam2.sa",
	SiteID:  "#Aflam2",
	Lang:    "ar",
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := artonline.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "الراقصه و السياسي" {
		t.Fatalf("unexpected title: %q", p.Title)
	}

	expectedStart := time.Date(2022, 3, 3, 21, 30, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 3, 3, 23, 4, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	expectedImage := "https://www.artonline.tv/UploadImages/Channel/ARTAFLAM1/03/AlRaqesaWaAlSeyasi.jpg"
	if p.Image != expectedImage {
		t.Fatalf("expected image %q, got %q", expectedImage, p.Image)
	}

	if p.Channel != "ARTAflam2.sa" {
		t.Fatalf("expected channel 'ARTAflam2.sa', got %q", p.Channel)
	}

	if p.Source != "artonline.tv" {
		t.Fatalf("expected source 'artonline.tv', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := artonline.Parse([]byte{}, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := artonline.Parse([]byte("not json"), channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
