package telecablesat_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/telecablesat"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(testdataPath(name))
	if err != nil {
		t.Fatalf("failed to read test data %s: %v", name, err)
	}
	return data
}

func TestParseContent(t *testing.T) {
	pages := [][]byte{
		readTestdata(t, "content_morning.html"),
		readTestdata(t, "content_noon.html"),
		readTestdata(t, "content_afternoon.html"),
	}

	channel := domain.Channel{
		ID:      "tv-programme.telecablesat.fr-13",
		XmltvID: "DasErste.de@SD",
		SiteID:  "13",
		Lang:    "de",
	}

	date := time.Date(2023, 11, 30, 0, 0, 0, 0, time.UTC)
	programs, err := telecablesat.Parse(pages, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Tagesschau" {
		t.Fatalf("expected title 'Tagesschau', got %q", p.Title)
	}

	if p.Description != "Die Tagesschau ist eine Institution in der deutschen Fernsehlandschaft. Seit 1952 wird kurz und b\u00fcndig von aktuellen Geschehnissen in Deutschland und der Welt berichtet. Bis heute ist die Redaktion der sachlichen Berichterstattung treu geblieben und..." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	expectedImage := "https://tv.cdnartwhere.eu/cache/i2/Dc5BDoMgEADAv3Cuwoqy4Fu4LLho24hEaNK06d_rcW7zFYEqi1lsrZU6e-nlXrqyHe2oXVxyT5_Xybys3GduXsYjN7pnPqdkI0CkJbk4gnKWMQFAQLQUtHZesuEwOgWa7DCkKV4cGFEBG0eQrCJSwY3YP8oqbmKn-rwexuBb20n8_g.jpg"
	if p.Image != expectedImage {
		t.Fatalf("expected image %q, got %q", expectedImage, p.Image)
	}

	expectedStart := time.Date(2023, 11, 30, 8, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 11, 30, 8, 5, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "DasErste.de@SD" {
		t.Fatalf("expected channel 'DasErste.de@SD', got %q", p.Channel)
	}

	if p.Source != "tv-programme.telecablesat.fr" {
		t.Fatalf("expected source 'tv-programme.telecablesat.fr', got %q", p.Source)
	}

	if p.Lang != "de" {
		t.Fatalf("expected lang 'de', got %q", p.Lang)
	}

	last := programs[len(programs)-1]

	if last.Title != "ZDF-Morgenmagazin" {
		t.Fatalf("expected last title 'ZDF-Morgenmagazin', got %q", last.Title)
	}

	if last.Description != "F\u00fcr einen guten Start in den Tag" {
		t.Fatalf("unexpected last description: %q", last.Description)
	}

	lastExpectedStart := time.Date(2023, 12, 1, 4, 30, 0, 0, time.UTC)
	if !last.Start.Equal(lastExpectedStart) {
		t.Fatalf("expected last start %v, got %v", lastExpectedStart, last.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	pages := [][]byte{
		[]byte(`<!DOCTYPE html><html lang="fr" dir="ltr" prefix=""> <head></head> <body></body></html>`),
	}

	channel := domain.Channel{
		XmltvID: "DasErste.de@SD",
		Lang:    "de",
	}

	date := time.Date(2023, 11, 30, 0, 0, 0, 0, time.UTC)
	programs, err := telecablesat.Parse(pages, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNoPages(t *testing.T) {
	programs, err := telecablesat.Parse(nil, domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseMorningOnly(t *testing.T) {
	pages := [][]byte{
		readTestdata(t, "content_morning.html"),
	}

	channel := domain.Channel{
		XmltvID: "DasErste.de@SD",
		SiteID:  "13",
		Lang:    "de",
	}

	date := time.Date(2023, 11, 30, 0, 0, 0, 0, time.UTC)
	programs, err := telecablesat.Parse(pages, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) == 0 {
		t.Fatal("expected at least one program from morning data")
	}
}
