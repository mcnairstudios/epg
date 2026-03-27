package melita_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/melita"
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
		ID:      "melita-tvm",
		XmltvID: "TVM.mt",
		SiteID:  "4d40a9f9-12fd-4f03-8072-61c637ff6995",
		Lang:    "en",
	}

	programs, err := melita.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "How I Met Your Mother" {
		t.Fatalf("expected title 'How I Met Your Mother', got %q", p.Title)
	}

	if p.Season != 7 {
		t.Fatalf("expected season 7, got %d", p.Season)
	}

	if p.Episode != 12 {
		t.Fatalf("expected episode 12, got %d", p.Episode)
	}

	if p.Category != "comedy" {
		t.Fatalf("expected category 'comedy', got %q", p.Category)
	}

	expectedStart := time.Date(2022, 4, 20, 6, 25, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 4, 20, 6, 45, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image != "https://androme.melitacable.com/media/images/epg/bc/07/p8953134_e_h10_ad.jpg?form=epg-card-6" {
		t.Fatalf("unexpected image: %q", p.Image)
	}
}

func TestParseEmpty(t *testing.T) {
	programs, err := melita.Parse([]byte("{}"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := melita.Parse([]byte("not json"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse should not error, got: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
