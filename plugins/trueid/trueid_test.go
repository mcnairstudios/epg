package trueid_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/trueid"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseEnglish(t *testing.T) {
	data, err := os.ReadFile(testdataPath("data.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "true-movie-hits",
		XmltvID: "TrueMovieHits.th",
		Lang:    "en",
	}

	programs, err := trueid.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "The Last Witch Hunter" {
		t.Fatalf("expected title 'The Last Witch Hunter', got %q", p.Title)
	}

	if p.Description != "A young man is all that stands between humanity and the most horrifying witches in history." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	if p.Image != "https://bms.dmpcdn.com/uploads/pic/381f853da5f4a310bf248357fed21a57.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	expectedStart := time.Date(2023, 12, 11, 19, 5, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 12, 11, 20, 55, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseThai(t *testing.T) {
	data, err := os.ReadFile(testdataPath("data.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "true-movie-hits",
		XmltvID: "TrueMovieHits.th",
		Lang:    "th",
	}

	programs, err := trueid.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Description == "" {
		t.Fatal("expected Thai description to be set")
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := trueid.Parse([]byte("{}"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := trueid.Parse([]byte("not json"), domain.Channel{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseBuildID(t *testing.T) {
	html := []byte(`<html><head></head><body><script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":{}},"page":"/","buildId":"abc123def456"}</script></body></html>`)
	buildID, err := trueid.ParseBuildID(html)
	if err != nil {
		t.Fatalf("parse build ID failed: %v", err)
	}
	if buildID != "abc123def456" {
		t.Fatalf("expected build ID 'abc123def456', got %q", buildID)
	}
}

func TestParseBuildIDMissing(t *testing.T) {
	html := []byte(`<html><head></head><body>no next data here</body></html>`)
	_, err := trueid.ParseBuildID(html)
	if err == nil {
		t.Fatal("expected error for missing __NEXT_DATA__")
	}
}

func TestParseBuildIDNoScriptEnd(t *testing.T) {
	html := []byte(`<html><body>{"props":{"pageProps":{}},"buildId":"abc123"}`)
	_, err := trueid.ParseBuildID(html)
	if err == nil {
		t.Fatal("expected error for missing </script> end tag")
	}
}
