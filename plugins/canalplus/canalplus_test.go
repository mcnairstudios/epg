package canalplus_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/canalplus"
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
		ID:      "canalplus-198",
		XmltvID: "CanalPlusCinemaFrance.fr",
		SiteID:  "bi#198",
		Lang:    "fr",
	}

	programs, err := canalplus.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Le cercle" {
		t.Fatalf("expected title 'Le cercle', got %q", p.Title)
	}

	expectedStart := time.Date(2023, 1, 12, 6, 28, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 1, 12, 12, 6, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "CanalPlusCinemaFrance.fr" {
		t.Fatalf("expected channel 'CanalPlusCinemaFrance.fr', got %q", p.Channel)
	}

	if p.Source != "canalplus.com" {
		t.Fatalf("expected source 'canalplus.com', got %q", p.Source)
	}
}

func TestParseDetail(t *testing.T) {
	data, err := os.ReadFile(testdataPath("program2.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	info, err := canalplus.ParseDetail(data)
	if err != nil {
		t.Fatalf("parse detail failed: %v", err)
	}

	if info == nil {
		t.Fatal("expected non-nil detail info")
	}

	if info.Image != "https://thumb.canalplus.pro/http/unsafe/{resolutionXY}/filters:quality({imageQualityPercentage})/img-hapi.canalplus.pro:80/ServiceImage/ImageID/107356485" {
		t.Fatalf("unexpected image: %q", info.Image)
	}

	if len(info.Directors) != 1 || info.Directors[0] != "Xavier Giannoli" {
		t.Fatalf("unexpected directors: %v", info.Directors)
	}

	if len(info.Actors) != 12 {
		t.Fatalf("expected 12 actors, got %d", len(info.Actors))
	}

	if info.Actors[0] != "Benjamin Voisin" {
		t.Fatalf("expected first actor 'Benjamin Voisin', got %q", info.Actors[0])
	}

	if len(info.Writers) != 1 || info.Writers[0] != "Xavier Giannoli" {
		t.Fatalf("unexpected writers: %v", info.Writers)
	}

	if info.Rating == nil {
		t.Fatal("expected rating to be set")
	}

	if info.Rating.System != "CSA" || info.Rating.Value != "-10" {
		t.Fatalf("expected rating CSA/-10, got %s/%s", info.Rating.System, info.Rating.Value)
	}
}

func TestParseDetailPresenter(t *testing.T) {
	data, err := os.ReadFile(testdataPath("program1.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	info, err := canalplus.ParseDetail(data)
	if err != nil {
		t.Fatalf("parse detail failed: %v", err)
	}

	if info == nil {
		t.Fatal("expected non-nil detail info")
	}

	if len(info.Presenters) != 1 || info.Presenters[0] != "Lily Bloom" {
		t.Fatalf("unexpected presenters: %v", info.Presenters)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := canalplus.Parse(data, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := canalplus.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
