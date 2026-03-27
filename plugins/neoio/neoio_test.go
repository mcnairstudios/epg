package neoio_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/neoio"
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
		ID:      "neo-tv-slo-1",
		XmltvID: "TVSLO1.si",
		SiteID:  "tv-slo-1",
		Lang:    "sl",
	}

	programs, err := neoio.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 3 {
		t.Fatalf("expected 3 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Napovedujemo" {
		t.Fatalf("expected title 'Napovedujemo', got %q", p.Title)
	}

	if p.Description != "Vabilo k ogledu naših oddaj." {
		t.Fatalf("expected description 'Vabilo k ogledu naših oddaj.', got %q", p.Description)
	}

	expectedStart := time.Date(2024, 12, 26, 4, 5, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2024, 12, 26, 5, 50, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image != "https://ngimg.siol.tv/sioltv/mtcmsprod/52/0/0/5200d01a-fe5f-487e-835a-274e77227a6b.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	p2 := programs[1]
	if p2.Title != "S0E0 - Hrabri zajčki: Prvi sneg" {
		t.Fatalf("expected second title, got %q", p2.Title)
	}

	expectedStart2 := time.Date(2024, 12, 26, 5, 50, 0, 0, time.UTC)
	if !p2.Start.Equal(expectedStart2) {
		t.Fatalf("expected start %v, got %v", expectedStart2, p2.Start)
	}

	expectedStop2 := time.Date(2024, 12, 26, 6, 0, 0, 0, time.UTC)
	if !p2.Stop.Equal(expectedStop2) {
		t.Fatalf("expected stop %v, got %v", expectedStop2, p2.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := neoio.Parse(data, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := neoio.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
