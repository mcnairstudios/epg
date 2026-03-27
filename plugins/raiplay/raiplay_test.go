package raiplay_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/raiplay"
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
		ID:      "raiplay-rai-2",
		XmltvID: "Rai2.it",
		SiteID:  "rai-2",
		Lang:    "it",
	}

	date := time.Date(2022, 5, 3, 0, 0, 0, 0, time.UTC)
	programs, err := raiplay.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "The Good Doctor S3E5 - La prima volta" {
		t.Fatalf("expected title 'The Good Doctor S3E5 - La prima volta', got %q", p.Title)
	}

	expectedStart := time.Date(2022, 5, 3, 17, 40, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 5, 3, 18, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.SubTitle != "La prima volta" {
		t.Fatalf("expected subtitle 'La prima volta', got %q", p.SubTitle)
	}

	if p.Season != 3 {
		t.Fatalf("expected season 3, got %d", p.Season)
	}

	if p.Episode != 5 {
		t.Fatalf("expected episode 5, got %d", p.Episode)
	}

	if p.Image != "https://www.raiplay.it/dl/img/2020/03/09/1583748471860_dddddd.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2022, 5, 3, 0, 0, 0, 0, time.UTC)
	programs, err := raiplay.Parse(data, domain.Channel{}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	date := time.Date(2022, 5, 3, 0, 0, 0, 0, time.UTC)
	programs, err := raiplay.Parse([]byte(`not json`), domain.Channel{}, date)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyBody(t *testing.T) {
	date := time.Date(2022, 5, 3, 0, 0, 0, 0, time.UTC)
	programs, err := raiplay.Parse([]byte{}, domain.Channel{}, date)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseHTMLResponse(t *testing.T) {
	date := time.Date(2022, 5, 3, 0, 0, 0, 0, time.UTC)
	html := []byte(`<!DOCTYPE html><html><body>Not Found</body></html>`)
	programs, err := raiplay.Parse(html, domain.Channel{}, date)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
