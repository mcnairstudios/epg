package tccuy_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tccuy"
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
		ID:      "programacion.tcc.com.uy-212",
		XmltvID: "MultiPremier.mx",
		SiteID:  "212",
		Lang:    "es",
	}

	programs, err := tccuy.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Meurtres à... - Temp. 3 - Episodio 3" {
		t.Fatalf("expected title 'Meurtres à... - Temp. 3 - Episodio 3', got %q", p.Title)
	}

	expectedStart := time.Date(2023, 2, 10, 22, 45, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 2, 11, 0, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Season != 3 {
		t.Fatalf("expected season 3, got %d", p.Season)
	}

	if p.Episode != 3 {
		t.Fatalf("expected episode 3, got %d", p.Episode)
	}

	if p.Image != "https://zpapi.zetatv.com.uy/media/images/2b45d2675389f2e4f7f6fe0655ccc968.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	p2 := programs[1]
	if p2.Title != "Grandes esperanzas" {
		t.Fatalf("expected title 'Grandes esperanzas', got %q", p2.Title)
	}

	if p2.Category != "Drama" {
		t.Fatalf("expected category 'Drama', got %q", p2.Category)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{SiteID: "212"}
	programs, err := tccuy.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse should not fail: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := tccuy.Parse([]byte("not json"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse should not fail: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
