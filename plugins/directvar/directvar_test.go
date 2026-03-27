package directvar_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/directvar"
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
		ID:      "directvar-207",
		XmltvID: "AEHDSouth.us",
		SiteID:  "207#A&amp;EHD",
		Lang:    "es",
	}

	programs, err := directvar.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Chicas guapas" {
		t.Fatalf("expected title 'Chicas guapas', got %q", p.Title)
	}

	expectedDesc := "Un espacio destinado a la belleza y los distintos estilos de vida, que muestra el trabajo inspiracional de la moda latinoamericana."
	if p.Description != expectedDesc {
		t.Fatalf("expected description %q, got %q", expectedDesc, p.Description)
	}

	expectedStart := time.Date(2022, 6, 19, 3, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 6, 19, 3, 15, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "AEHDSouth.us" {
		t.Fatalf("expected channel 'AEHDSouth.us', got %q", p.Channel)
	}

	if p.Source != "directv.com.ar" {
		t.Fatalf("expected source 'directv.com.ar', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	channel := domain.Channel{
		SiteID: "207#A&amp;EHD",
	}
	programs, err := directvar.Parse([]byte{}, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	channel := domain.Channel{
		SiteID: "207#A&amp;EHD",
	}
	_, err := directvar.Parse([]byte(`not json`), channel)
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseNoPrograms(t *testing.T) {
	channel := domain.Channel{
		SiteID: "999#NoChannel",
	}
	programs, err := directvar.Parse([]byte(`{"d":[]}`), channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
