package knrgl_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/knrgl"
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

	date := time.Date(2021, 11, 22, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		ID:      "knrgl-knr",
		XmltvID: "KNR.gl",
		SiteID:  "#",
		Lang:    "kl",
	}

	programs, err := knrgl.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Issittormiuaqqat" {
		t.Fatalf("expected title 'Issittormiuaqqat', got %q", p.Title)
	}

	expectedStart := time.Date(2021, 11, 22, 11, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2021, 11, 22, 11, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if len(programs) > 4 {
		p4 := programs[4]
		if p4.Title != "KNR2: Tusagassiortunik katersortitsineq - Erik Jensen" {
			t.Fatalf("expected 5th title, got %q", p4.Title)
		}

		expectedStart4 := time.Date(2021, 11, 22, 13, 0, 0, 0, time.UTC)
		if !p4.Start.Equal(expectedStart4) {
			t.Fatalf("expected 5th start %v, got %v", expectedStart4, p4.Start)
		}

		expectedStop4 := time.Date(2021, 11, 22, 13, 30, 0, 0, time.UTC)
		if !p4.Stop.Equal(expectedStop4) {
			t.Fatalf("expected 5th stop %v, got %v", expectedStop4, p4.Stop)
		}
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2021, 11, 22, 0, 0, 0, 0, time.UTC)
	programs, err := knrgl.Parse(data, domain.Channel{}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	date := time.Date(2021, 11, 22, 0, 0, 0, 0, time.UTC)
	_, err := knrgl.Parse([]byte(`not json`), domain.Channel{}, date)
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
