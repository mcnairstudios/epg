package elpais_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/elpais"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	gridData, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read grid test data: %v", err)
	}

	programsData, err := os.ReadFile(testdataPath("programs.json"))
	if err != nil {
		t.Fatalf("failed to read programs test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "programacion-tv.elpais.com-3",
		XmltvID: "La1.es",
		SiteID:  "3",
		Lang:    "es",
	}

	programs, err := elpais.Parse(gridData, programsData, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Comerse el mundo" {
		t.Fatalf("expected title 'Comerse el mundo', got %q", p.Title)
	}

	if p.SubTitle != "Par\u00eds" {
		t.Fatalf("expected subtitle 'Par\u00eds', got %q", p.SubTitle)
	}

	if p.Description != "El chef Pe\u00f1a viaja hasta Par\u00eds, una de las capitales mundiales de la alta gastronom\u00eda. All\u00ed visitar\u00e1 un vi\u00f1edo muy especial en pleno coraz\u00f3n de la ciudad, probar\u00e1 los famosos caracoles, har\u00e1 un queso y conocer\u00e1 a chefs que llegaron a la capital gala para cumplir sus sue\u00f1os y los consiguieron." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	if p.Category != "Ocio-Cultura/Cocina" {
		t.Fatalf("expected category 'Ocio-Cultura/Cocina', got %q", p.Category)
	}

	if p.Season != 1 {
		t.Fatalf("expected season 1, got %d", p.Season)
	}

	if p.Episode != 23 {
		t.Fatalf("expected episode 23, got %d", p.Episode)
	}

	if p.Image != "https://programacion-tv.elpais.com/imagenes/programas/2099957.jpg" {
		t.Fatalf("expected image URL, got %q", p.Image)
	}

	madrid, _ := time.LoadLocation("Europe/Madrid")
	expectedStart := time.Date(2022, 10, 4, 1, 30, 0, 0, madrid)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 10, 4, 2, 25, 0, 0, madrid)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "La1.es" {
		t.Fatalf("expected channel 'La1.es', got %q", p.Channel)
	}

	if p.Source != "programacion-tv.elpais.com" {
		t.Fatalf("expected source 'programacion-tv.elpais.com', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := elpais.Parse(nil, nil, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := elpais.Parse([]byte(`not json`), nil, domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseNoMatchingChannel(t *testing.T) {
	gridData := []byte(`[{"idCanal":"999","programas":[]}]`)
	channel := domain.Channel{SiteID: "1"}

	programs, err := elpais.Parse(gridData, nil, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNoProgramDetails(t *testing.T) {
	gridData, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read grid test data: %v", err)
	}

	channel := domain.Channel{
		XmltvID: "La1.es",
		SiteID:  "3",
		Lang:    "es",
	}

	programs, err := elpais.Parse(gridData, nil, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Comerse el mundo" {
		t.Fatalf("expected title 'Comerse el mundo', got %q", p.Title)
	}

	if p.Description != "El cocinero Javier Pe\u00f1a recorre diversos pa\u00edses mostr\u00e1ndonos su diversidad gastron\u00f3mica." {
		t.Fatalf("expected grid description as fallback, got %q", p.Description)
	}
}
