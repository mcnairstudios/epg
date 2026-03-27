package skyit_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/skyit"
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
		ID:      "guidatv.sky.it-DTH#10458",
		XmltvID: "20Mediaset.it",
		SiteID:  "DTH#10458",
		Lang:    "it",
	}

	programs, err := skyit.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Distretto di Polizia" {
		t.Fatalf("expected title 'Distretto di Polizia', got %q", p.Title)
	}

	expectedDesc := "S6 Ep26 La resa dei conti - Fino all'ultimo la sfida tra Ardenzi e Carrano, nemici di vecchia data, riserva clamorosi colpi di scena. E si scopre che non e' tutto come sembrava."
	if p.Description != expectedDesc {
		t.Fatalf("expected description %q, got %q", expectedDesc, p.Description)
	}

	expectedStart := time.Date(2022, 5, 6, 0, 35, 40, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 5, 6, 1, 15, 40, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Season != 6 {
		t.Fatalf("expected season 6, got %d", p.Season)
	}

	if p.Episode != 26 {
		t.Fatalf("expected episode 26, got %d", p.Episode)
	}

	if p.Category != "Intrattenimento/Fiction" {
		t.Fatalf("expected category 'Intrattenimento/Fiction', got %q", p.Category)
	}

	expectedImage := "https://guidatv.sky.it/uuid/77c630aa-4744-44cb-a88e-3e871c6b73d9/cover?md5ChecksumParam=61135b999a63e3d3f4a933b9edeb0c1b"
	if p.Image != expectedImage {
		t.Fatalf("expected image %q, got %q", expectedImage, p.Image)
	}

	if p.Channel != "20Mediaset.it" {
		t.Fatalf("expected channel '20Mediaset.it', got %q", p.Channel)
	}

	if p.Source != "guidatv.sky.it" {
		t.Fatalf("expected source 'guidatv.sky.it', got %q", p.Source)
	}

	if p.Lang != "it" {
		t.Fatalf("expected lang 'it', got %q", p.Lang)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := skyit.Parse(data, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := skyit.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseNoEvents(t *testing.T) {
	programs, err := skyit.Parse([]byte(`{}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseSeasonTooLarge(t *testing.T) {
	data := []byte(`{"events":[{"eventTitle":"Test","starttime":"2022-05-06T00:00:00Z","endtime":"2022-05-06T01:00:00Z","content":{"seasonNumber":123,"episodeNumber":1,"genre":{},"subgenre":{}}}]}`)
	programs, err := skyit.Parse(data, domain.Channel{Lang: "it"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if programs[0].Season != 0 {
		t.Fatalf("expected season 0 for large season number, got %d", programs[0].Season)
	}
}

func TestParseEpisodeTooLarge(t *testing.T) {
	data := []byte(`{"events":[{"eventTitle":"Test","starttime":"2022-05-06T00:00:00Z","endtime":"2022-05-06T01:00:00Z","content":{"seasonNumber":1,"episodeNumber":1234,"genre":{},"subgenre":{}}}]}`)
	programs, err := skyit.Parse(data, domain.Channel{Lang: "it"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if programs[0].Episode != 0 {
		t.Fatalf("expected episode 0 for large episode number, got %d", programs[0].Episode)
	}
}
