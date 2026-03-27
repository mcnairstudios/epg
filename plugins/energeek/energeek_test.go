package energeek_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/energeek"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

var channel = domain.Channel{
	ID:      "energeek.cl-EnerGeek Retro",
	XmltvID: "EnerGeekRetro.cl",
	SiteID:  "EnerGeek Retro",
	Lang:    "es",
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.xml"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2022, 11, 29, 0, 0, 0, 0, time.UTC)
	programs, err := energeek.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Noir" {
		t.Fatalf("expected title 'Noir', got %q", p.Title)
	}

	expectedStart := time.Date(2022, 11, 29, 3, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 11, 29, 3, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image != "https://pics.filmaffinity.com/nowaru_noir_tv_series-225888552-mmed.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	if p.Channel != "EnerGeekRetro.cl" {
		t.Fatalf("expected channel 'EnerGeekRetro.cl', got %q", p.Channel)
	}

	if p.Source != "energeek.cl" {
		t.Fatalf("expected source 'energeek.cl', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := energeek.Parse([]byte{}, channel, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
