package mytelly_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/mytelly"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseListing(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	urls := mytelly.ParseListing(data)
	if len(urls) != 25 {
		t.Fatalf("expected 25 programme URLs, got %d", len(urls))
	}

	expected := "https://www.mytelly.co.uk/tv-guide/listings/programme?cid=713&pid=1906433&tm=2024-12-07+00%3A00%3A00"
	if urls[0] != expected {
		t.Fatalf("expected first URL %q, got %q", expected, urls[0])
	}
}

func TestParseProgramme(t *testing.T) {
	data, err := os.ReadFile(testdataPath("programme.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "713/bbc-one-london",
		XmltvID: "BBCOneLondon.uk",
		Lang:    "en",
	}

	date := time.Date(2024, 12, 7, 0, 0, 0, 0, time.UTC)

	prog := mytelly.ParseProgramme(data, channel, date)

	if prog.Title != "Captain Phillips" {
		t.Fatalf("expected title %q, got %q", "Captain Phillips", prog.Title)
	}

	if prog.Start.UTC().Format("2006-01-02T15:04:05.000Z") != "2024-12-07T00:00:00.000Z" {
		t.Fatalf("expected start 2024-12-07T00:00:00.000Z, got %s", prog.Start.UTC().Format("2006-01-02T15:04:05.000Z"))
	}

	if prog.Stop.UTC().Format("2006-01-02T15:04:05.000Z") != "2024-12-07T02:05:00.000Z" {
		t.Fatalf("expected stop 2024-12-07T02:05:00.000Z, got %s", prog.Stop.UTC().Format("2006-01-02T15:04:05.000Z"))
	}

	expectedDesc := "An American cargo ship sets a dangerous course around the coast of Somalia, while inland, four men are pressed into service as pirates by the local warlords. The captain is taken hostage when the raiding party hijacks the vessel, resulting in a tense five-day crisis. Fact-based thriller, starring Tom Hanks and Barkhad Abdi"
	if prog.Description != expectedDesc {
		t.Fatalf("expected description %q, got %q", expectedDesc, prog.Description)
	}

	expectedImage := "https://d16ia5iwuvax6y.cloudfront.net/uk-prog-images/c44ce7b0d3ae602c0c93ece5af140815.jpg?k=VeeNdUjml3bSHdlZ0OXbGLy%2BmsLdYPwTV6iAxGkzq4dsylOCGGE7OWlqwSWt0cd0Qtrin4DkEMC0Zzdp8ZeNk2vNIQzjMF0DG0h3IeTR5NM%3D"
	if prog.Image != expectedImage {
		t.Fatalf("expected image %q, got %q", expectedImage, prog.Image)
	}

	if prog.Category != "Factual,Movie/Drama,Thriller" {
		t.Fatalf("expected category %q, got %q", "Factual,Movie/Drama,Thriller", prog.Category)
	}
}

func TestParseProgrammeWithSeasonEpisode(t *testing.T) {
	data, err := os.ReadFile(testdataPath("programme2.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "713/bbc-one-london",
		XmltvID: "BBCOneLondon.uk",
		Lang:    "en",
	}

	date := time.Date(2024, 12, 7, 0, 0, 0, 0, time.UTC)

	prog := mytelly.ParseProgramme(data, channel, date)

	if prog.Title != "The Rap Game UK" {
		t.Fatalf("expected title %q, got %q", "The Rap Game UK", prog.Title)
	}

	if prog.Start.UTC().Format("2006-01-02T15:04:05.000Z") != "2024-12-07T23:35:00.000Z" {
		t.Fatalf("expected start 2024-12-07T23:35:00.000Z, got %s", prog.Start.UTC().Format("2006-01-02T15:04:05.000Z"))
	}

	if prog.Stop.UTC().Format("2006-01-02T15:04:05.000Z") != "2024-12-08T00:40:00.000Z" {
		t.Fatalf("expected stop 2024-12-08T00:40:00.000Z, got %s", prog.Stop.UTC().Format("2006-01-02T15:04:05.000Z"))
	}

	expectedSubTitle := "Past and Pressure Season 6, Episode 5"
	if prog.SubTitle != expectedSubTitle {
		t.Fatalf("expected subtitle %q, got %q", expectedSubTitle, prog.SubTitle)
	}

	if prog.Season != 6 {
		t.Fatalf("expected season 6, got %d", prog.Season)
	}

	if prog.Episode != 5 {
		t.Fatalf("expected episode 5, got %d", prog.Episode)
	}

	if prog.Category != "Challenge/Reality Show,Show/Game Show" {
		t.Fatalf("expected category %q, got %q", "Challenge/Reality Show,Show/Game Show", prog.Category)
	}
}

func TestParseProgrammeEmpty(t *testing.T) {
	data := []byte(`<!DOCTYPE html><html><head></head><body></body></html>`)
	channel := domain.Channel{
		SiteID:  "713/bbc-one-london",
		XmltvID: "BBCOneLondon.uk",
		Lang:    "en",
	}

	date := time.Date(2024, 12, 7, 0, 0, 0, 0, time.UTC)

	prog := mytelly.ParseProgramme(data, channel, date)
	if prog.Title != "" {
		t.Fatalf("expected empty title, got %q", prog.Title)
	}
}

func TestParseListingEmpty(t *testing.T) {
	data := []byte(`<!DOCTYPE html><html><head></head><body></body></html>`)
	urls := mytelly.ParseListing(data)
	if len(urls) != 0 {
		t.Fatalf("expected 0 URLs, got %d", len(urls))
	}
}
