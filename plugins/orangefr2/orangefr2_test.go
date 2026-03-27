package orangefr2_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/orangefr2"
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
		ID:      "chaines-tv.orange.fr-192",
		XmltvID: "TF1.fr",
		SiteID:  "192",
		Lang:    "fr",
	}

	programs, err := orangefr2.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Tête de liste" {
		t.Fatalf("expected title 'Tête de liste', got %q", p.Title)
	}

	if p.SubTitle != "Esprits criminels" {
		t.Fatalf("expected subtitle 'Esprits criminels', got %q", p.SubTitle)
	}

	expectedStart := time.Date(2021, 11, 7, 23, 35, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2021, 11, 8, 0, 20, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Season != 10 {
		t.Fatalf("expected season 10, got %d", p.Season)
	}

	if p.Episode != 12 {
		t.Fatalf("expected episode 12, got %d", p.Episode)
	}

	if p.Category != "Série Suspense" {
		t.Fatalf("expected category 'Série Suspense', got %q", p.Category)
	}

	if p.Channel != "TF1.fr" {
		t.Fatalf("expected channel 'TF1.fr', got %q", p.Channel)
	}

	if p.Source != "chaines-tv.orange.fr" {
		t.Fatalf("expected source 'chaines-tv.orange.fr', got %q", p.Source)
	}

	if p.Image != "https://proxymedia.woopic.com/340/p/169_EMI_9697669.jpg" {
		t.Fatalf("expected image URL, got %q", p.Image)
	}

	expectedDesc := "Un tueur en série prend un plaisir pervers à prévenir les autorités de Tallahassee avant chaque nouveau meurtre. Rossi apprend le décès d'un de ses vieux amis."
	if p.Description != expectedDesc {
		t.Fatalf("expected description %q, got %q", expectedDesc, p.Description)
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID: "192",
	}

	programs, err := orangefr2.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := orangefr2.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatal("expected nil error for invalid JSON (graceful handling)")
	}
	if programs != nil {
		t.Fatalf("expected nil programs, got %v", programs)
	}
}

func TestParseEmptyChannelData(t *testing.T) {
	programs, err := orangefr2.Parse([]byte(`{"999":[]}`), domain.Channel{SiteID: "192"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if programs != nil {
		t.Fatalf("expected nil programs, got %v", programs)
	}
}
