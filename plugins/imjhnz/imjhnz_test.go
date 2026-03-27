package imjhnz_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/imjhnz"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.xml"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "imjhnz-plex-darkmatter",
		XmltvID: "DarkMatterTV.us",
		SiteID:  "Plex/all#5e20b730f2f8d5003d739db7-5eea605674085f0040ddc7a6",
		Lang:    "en",
	}
	channelID := "5e20b730f2f8d5003d739db7-5eea605674085f0040ddc7a6"
	date := time.Date(2023, 6, 23, 0, 0, 0, 0, time.UTC)

	programs, err := imjhnz.Parse(data, channel, channelID, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Killers Within" {
		t.Fatalf("expected title 'Killers Within', got %q", p.Title)
	}

	if p.Category != "Movie" {
		t.Fatalf("expected category 'Movie', got %q", p.Category)
	}

	if p.Channel != "DarkMatterTV.us" {
		t.Fatalf("expected channel 'DarkMatterTV.us', got %q", p.Channel)
	}

	if p.Source != "i.mjh.nz" {
		t.Fatalf("expected source 'i.mjh.nz', got %q", p.Source)
	}

	expectedStart := time.Date(2023, 6, 23, 7, 14, 32, 0, time.UTC)
	if !p.Start.UTC().Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start.UTC())
	}

	if p.Description == "" {
		t.Fatal("expected non-empty description")
	}

	if p.Image == "" {
		t.Fatal("expected non-empty image")
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := imjhnz.Parse(
		[]byte(`<?xml version="1.0" encoding="UTF-8"?><tv></tv>`),
		domain.Channel{XmltvID: "test"},
		"ch1",
		time.Now(),
	)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidXML(t *testing.T) {
	_, err := imjhnz.Parse(
		[]byte(`not xml`),
		domain.Channel{},
		"ch1",
		time.Now(),
	)
	if err == nil {
		t.Fatal("expected error on invalid XML")
	}
}

func TestMovieMerging(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tv>
  <programme channel="ch1" start="20230623100000 +0000" stop="20230623110000 +0000">
    <title>Test Movie</title>
    <desc>A test movie</desc>
    <category>Movie</category>
  </programme>
  <programme channel="ch1" start="20230623110000 +0000" stop="20230623120000 +0000">
    <title>Test Movie</title>
    <desc>A test movie</desc>
    <category>Movie</category>
  </programme>
  <programme channel="ch1" start="20230623120000 +0000" stop="20230623130000 +0000">
    <title>Different Show</title>
    <desc>Something else</desc>
    <category>Drama</category>
  </programme>
</tv>`)

	programs, err := imjhnz.Parse(
		xmlData,
		domain.Channel{XmltvID: "test"},
		"ch1",
		time.Date(2023, 6, 23, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs (merged movie + show), got %d", len(programs))
	}

	if programs[0].Title != "Test Movie" {
		t.Fatalf("expected 'Test Movie', got %q", programs[0].Title)
	}

	expectedStop := time.Date(2023, 6, 23, 12, 0, 0, 0, time.UTC)
	if !programs[0].Stop.Equal(expectedStop) {
		t.Fatalf("expected merged stop %v, got %v", expectedStop, programs[0].Stop)
	}
}
