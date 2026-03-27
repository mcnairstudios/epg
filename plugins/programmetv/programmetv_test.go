package programmetv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/programmetv"
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
		ID:      "programme-tv.net-tf1-19",
		XmltvID: "TF1.fr",
		SiteID:  "tf1-19",
		Lang:    "fr",
	}

	date := time.Date(2023, 11, 27, 0, 0, 0, 0, time.UTC)
	programs, err := programmetv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 28 {
		t.Fatalf("expected 28 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Programmes de la nuit" {
		t.Fatalf("expected title 'Programmes de la nuit', got %q", p.Title)
	}

	if p.Category != "Autre" {
		t.Fatalf("expected category 'Autre', got %q", p.Category)
	}

	expectedImage := "https://www.programme-tv.net/imgre/fit/~2~program~978eb86d5b99cee0.jpg/960x540/quality/80/programmes-de-la-nuit.jpg"
	if p.Image != expectedImage {
		t.Fatalf("expected image %q, got %q", expectedImage, p.Image)
	}

	expectedStart := time.Date(2023, 11, 27, 0, 5, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 11, 27, 5, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "TF1.fr" {
		t.Fatalf("expected channel 'TF1.fr', got %q", p.Channel)
	}

	if p.Source != "programme-tv.net" {
		t.Fatalf("expected source 'programme-tv.net', got %q", p.Source)
	}

	if p.Lang != "fr" {
		t.Fatalf("expected lang 'fr', got %q", p.Lang)
	}

	last := programs[27]

	if last.Title != "Coup de foudre chez le Père Noël" {
		t.Fatalf("expected title 'Coup de foudre chez le Père Noël', got %q", last.Title)
	}

	if last.Category != "Téléfilm" {
		t.Fatalf("expected category 'Téléfilm', got %q", last.Category)
	}

	lastExpectedImage := "https://www.programme-tv.net/imgre/fit/~2~program~5a4e78779c4a3fac.jpg/960x540/quality/80/coup-de-foudre-chez-le-pere-noel.jpg"
	if last.Image != lastExpectedImage {
		t.Fatalf("expected image %q, got %q", lastExpectedImage, last.Image)
	}

	lastExpectedStart := time.Date(2023, 11, 27, 22, 50, 0, 0, time.UTC)
	if !last.Start.Equal(lastExpectedStart) {
		t.Fatalf("expected start %v, got %v", lastExpectedStart, last.Start)
	}

	lastExpectedStop := time.Date(2023, 11, 27, 23, 45, 0, 0, time.UTC)
	if !last.Stop.Equal(lastExpectedStop) {
		t.Fatalf("expected stop %v, got %v", lastExpectedStop, last.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	channel := domain.Channel{
		XmltvID: "TF1.fr",
		Lang:    "fr",
	}
	date := time.Date(2023, 11, 27, 0, 0, 0, 0, time.UTC)

	programs, err := programmetv.Parse([]byte(""), channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNoCards(t *testing.T) {
	channel := domain.Channel{
		XmltvID: "TF1.fr",
		Lang:    "fr",
	}
	date := time.Date(2023, 11, 27, 0, 0, 0, 0, time.UTC)

	programs, err := programmetv.Parse([]byte("<html><body></body></html>"), channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
