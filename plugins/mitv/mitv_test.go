package mitv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/mitv"
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

	channel := domain.Channel{
		SiteID:  "ar#24-7-canal-de-noticias",
		XmltvID: "247CanaldeNoticias.ar",
		Lang:    "es",
	}
	date := time.Date(2021, 11, 24, 0, 0, 0, 0, time.UTC)

	programs, err := mitv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 3 {
		t.Fatalf("expected 3 programs, got %d", len(programs))
	}

	p0 := programs[0]
	if p0.Title != "Trasnoche de 24/7" {
		t.Fatalf("expected title 'Trasnoche de 24/7', got %q", p0.Title)
	}
	if p0.Category != "Interés general" {
		t.Fatalf("expected category 'Interés general', got %q", p0.Category)
	}
	if p0.Description != "Lo más visto de la semana en nuestra pantalla." {
		t.Fatalf("expected description 'Lo más visto de la semana en nuestra pantalla.', got %q", p0.Description)
	}
	if p0.Image != "https://cdn.mitvstatic.com/programs/fallback_other_l_m.jpg" {
		t.Fatalf("expected image URL, got %q", p0.Image)
	}
	expectedStart0 := time.Date(2021, 11, 24, 3, 0, 0, 0, time.UTC)
	if !p0.Start.Equal(expectedStart0) {
		t.Fatalf("expected start %v, got %v", expectedStart0, p0.Start)
	}
	expectedStop0 := time.Date(2021, 11, 24, 23, 0, 0, 0, time.UTC)
	if !p0.Stop.Equal(expectedStop0) {
		t.Fatalf("expected stop %v, got %v", expectedStop0, p0.Stop)
	}

	p1 := programs[1]
	if p1.Title != "Noticiero central - Segunda edición" {
		t.Fatalf("expected title 'Noticiero central - Segunda edición', got %q", p1.Title)
	}
	expectedStart1 := time.Date(2021, 11, 24, 23, 0, 0, 0, time.UTC)
	if !p1.Start.Equal(expectedStart1) {
		t.Fatalf("expected start %v, got %v", expectedStart1, p1.Start)
	}
	expectedStop1 := time.Date(2021, 11, 25, 1, 0, 0, 0, time.UTC)
	if !p1.Stop.Equal(expectedStop1) {
		t.Fatalf("expected stop %v, got %v", expectedStop1, p1.Stop)
	}

	p2 := programs[2]
	if p2.Title != "Plus energético" {
		t.Fatalf("expected title 'Plus energético', got %q", p2.Title)
	}
	expectedStart2 := time.Date(2021, 11, 25, 1, 0, 0, 0, time.UTC)
	if !p2.Start.Equal(expectedStart2) {
		t.Fatalf("expected start %v, got %v", expectedStart2, p2.Start)
	}
	expectedStop2 := time.Date(2021, 11, 25, 2, 0, 0, 0, time.UTC)
	if !p2.Stop.Equal(expectedStop2) {
		t.Fatalf("expected stop %v, got %v", expectedStop2, p2.Stop)
	}

	if p2.Channel != "247CanaldeNoticias.ar" {
		t.Fatalf("expected channel '247CanaldeNoticias.ar', got %q", p2.Channel)
	}
	if p2.Source != "mi.tv" {
		t.Fatalf("expected source 'mi.tv', got %q", p2.Source)
	}
	if p2.Lang != "es" {
		t.Fatalf("expected lang 'es', got %q", p2.Lang)
	}
}

func TestParseEmptyContent(t *testing.T) {
	channel := domain.Channel{SiteID: "ar#test", XmltvID: "Test.ar"}
	date := time.Date(2021, 11, 24, 0, 0, 0, 0, time.UTC)

	programs, err := mitv.Parse([]byte(""), channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{SiteID: "ar#test", XmltvID: "Test.ar"}
	date := time.Date(2021, 11, 24, 0, 0, 0, 0, time.UTC)

	programs, err := mitv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
