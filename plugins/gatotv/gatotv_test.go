package gatotv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/gatotv"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

var channel = domain.Channel{
	ID:      "gatotv-m_0",
	XmltvID: "0porMovistarPlus.es",
	SiteID:  "m_0",
	Lang:    "es",
}

func TestParseContent0(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_0.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2023, 6, 13, 0, 0, 0, 0, time.UTC)
	programs, err := gatotv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Supergarcía" {
		t.Fatalf("expected title 'Supergarcía', got %q", p.Title)
	}
	expectedStart := time.Date(2023, 6, 13, 4, 30, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
	expectedStop := time.Date(2023, 6, 13, 5, 32, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	p1 := programs[1]
	if p1.Title != "La resistencia" {
		t.Fatalf("expected title 'La resistencia', got %q", p1.Title)
	}
	expectedStart1 := time.Date(2023, 6, 13, 5, 32, 0, 0, time.UTC)
	if !p1.Start.Equal(expectedStart1) {
		t.Fatalf("expected start %v, got %v", expectedStart1, p1.Start)
	}
	expectedStop1 := time.Date(2023, 6, 13, 6, 59, 0, 0, time.UTC)
	if !p1.Stop.Equal(expectedStop1) {
		t.Fatalf("expected stop %v, got %v", expectedStop1, p1.Stop)
	}

	if len(programs) < 26 {
		t.Fatalf("expected at least 26 programs, got %d", len(programs))
	}

	p25 := programs[25]
	if p25.Title != "Una familia absolutamente normal" {
		t.Fatalf("expected title 'Una familia absolutamente normal', got %q", p25.Title)
	}
	expectedStart25 := time.Date(2023, 6, 14, 4, 46, 0, 0, time.UTC)
	if !p25.Start.Equal(expectedStart25) {
		t.Fatalf("expected start %v, got %v", expectedStart25, p25.Start)
	}
	expectedStop25 := time.Date(2023, 6, 14, 5, 0, 0, 0, time.UTC)
	if !p25.Stop.Equal(expectedStop25) {
		t.Fatalf("expected stop %v, got %v", expectedStop25, p25.Stop)
	}

	if p.Channel != "0porMovistarPlus.es" {
		t.Fatalf("expected channel '0porMovistarPlus.es', got %q", p.Channel)
	}
	if p.Source != "gatotv.com" {
		t.Fatalf("expected source 'gatotv.com', got %q", p.Source)
	}
	if p.Lang != "es" {
		t.Fatalf("expected lang 'es', got %q", p.Lang)
	}
}

func TestParseContent1MidnightStart(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_1.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2023, 6, 14, 0, 0, 0, 0, time.UTC)
	programs, err := gatotv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Ilustres Ignorantes" {
		t.Fatalf("expected title 'Ilustres Ignorantes', got %q", p.Title)
	}
	expectedStart := time.Date(2023, 6, 14, 5, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
	expectedStop := time.Date(2023, 6, 14, 5, 32, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if len(programs) < 27 {
		t.Fatalf("expected at least 27 programs, got %d", len(programs))
	}

	p26 := programs[26]
	if p26.Title != "Showriano" {
		t.Fatalf("expected title 'Showriano', got %q", p26.Title)
	}
	expectedStart26 := time.Date(2023, 6, 15, 4, 30, 0, 0, time.UTC)
	if !p26.Start.Equal(expectedStart26) {
		t.Fatalf("expected start %v, got %v", expectedStart26, p26.Start)
	}
	expectedStop26 := time.Date(2023, 6, 15, 5, 30, 0, 0, time.UTC)
	if !p26.Stop.Equal(expectedStop26) {
		t.Fatalf("expected stop %v, got %v", expectedStop26, p26.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := gatotv.Parse([]byte{}, channel, time.Now())
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

	programs, err := gatotv.Parse(data, channel, time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
