package xmltv_test

import (
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/xmltv"
)

func TestParse(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tv>
  <channel id="ch1">
    <display-name lang="en">Test Channel</display-name>
    <icon src="https://example.com/logo.png"/>
  </channel>
  <programme channel="ch1" start="20260327100000 +0000" stop="20260327110000 +0000">
    <title lang="en">Test Show</title>
    <desc lang="en">A test description</desc>
    <category lang="en">News</category>
    <episode-num system="onscreen">S02E05</episode-num>
  </programme>
</tv>`)

	channels, programs, err := xmltv.Parse(data, "test-source")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(channels))
	}

	ch := channels[0]
	if ch.Name != "Test Channel" {
		t.Fatalf("expected name 'Test Channel', got %q", ch.Name)
	}
	if ch.Logo != "https://example.com/logo.png" {
		t.Fatalf("expected logo, got %q", ch.Logo)
	}
	if ch.Source != "test-source" {
		t.Fatalf("expected source 'test-source', got %q", ch.Source)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Test Show" {
		t.Fatalf("expected title 'Test Show', got %q", p.Title)
	}
	if p.Description != "A test description" {
		t.Fatalf("expected description, got %q", p.Description)
	}
	if p.Category != "News" {
		t.Fatalf("expected category 'News', got %q", p.Category)
	}
	if p.Season != 2 {
		t.Fatalf("expected season 2, got %d", p.Season)
	}
	if p.Episode != 5 {
		t.Fatalf("expected episode 5, got %d", p.Episode)
	}

	expectedStart := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
}

func TestParseWithTimezone(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tv>
  <programme channel="ch1" start="20230623190000 +1200" stop="20230623200000 +1200">
    <title>Test</title>
  </programme>
</tv>`)

	_, programs, err := xmltv.Parse(data, "test")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	expectedStart := time.Date(2023, 6, 23, 7, 0, 0, 0, time.UTC)
	if !programs[0].Start.UTC().Equal(expectedStart) {
		t.Fatalf("expected start %v UTC, got %v", expectedStart, programs[0].Start.UTC())
	}
}

func TestParseXmltvNsEpisode(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<tv>
  <programme channel="ch1" start="20260327100000 +0000" stop="20260327110000 +0000">
    <title>Test</title>
    <episode-num system="xmltv_ns">1.79.</episode-num>
  </programme>
</tv>`)

	_, programs, err := xmltv.Parse(data, "test")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if programs[0].Season != 2 {
		t.Fatalf("expected season 2, got %d", programs[0].Season)
	}
	if programs[0].Episode != 80 {
		t.Fatalf("expected episode 80, got %d", programs[0].Episode)
	}
}

func TestParseInvalidXML(t *testing.T) {
	_, _, err := xmltv.Parse([]byte("not xml"), "test")
	if err == nil {
		t.Fatal("expected error on invalid XML")
	}
}

func TestParseEmpty(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?><tv></tv>`)
	channels, programs, err := xmltv.Parse(data, "test")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(channels) != 0 {
		t.Fatalf("expected 0 channels, got %d", len(channels))
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
