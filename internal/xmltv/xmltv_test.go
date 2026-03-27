package xmltv_test

import (
	"strings"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/xmltv"
)

func TestGenerate(t *testing.T) {
	channels := []domain.Channel{
		{ID: "ch1", XmltvID: "CNN.us", Name: "CNN", Lang: "en", Logo: "https://example.com/cnn.png"},
	}

	start := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	programs := []domain.Program{
		{
			Channel:     "CNN.us",
			Title:       "Breaking News",
			Description: "Latest headlines",
			Category:    "News",
			Lang:        "en",
			Start:       start,
			Stop:        start.Add(time.Hour),
		},
	}

	data, err := xmltv.Generate(channels, programs)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	xml := string(data)

	if !strings.Contains(xml, `<tv`) {
		t.Fatal("expected <tv> root element")
	}
	if !strings.Contains(xml, `id="CNN.us"`) {
		t.Fatal("expected channel id")
	}
	if !strings.Contains(xml, `<display-name lang="en">CNN</display-name>`) {
		t.Fatal("expected display name")
	}
	if !strings.Contains(xml, `src="https://example.com/cnn.png"`) {
		t.Fatal("expected icon")
	}
	if !strings.Contains(xml, `channel="CNN.us"`) {
		t.Fatal("expected programme channel attr")
	}
	if !strings.Contains(xml, "Breaking News") {
		t.Fatal("expected programme title")
	}
	if !strings.Contains(xml, "Latest headlines") {
		t.Fatal("expected programme description")
	}
}

func TestGenerateWithEpisodeInfo(t *testing.T) {
	channels := []domain.Channel{
		{ID: "ch1", XmltvID: "Naruto.us", Name: "Naruto", Lang: "en"},
	}

	start := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	programs := []domain.Program{
		{
			Channel: "Naruto.us",
			Title:   "The Third Hokage",
			Season:  2,
			Episode: 80,
			Lang:    "en",
			Start:   start,
			Stop:    start.Add(30 * time.Minute),
		},
	}

	data, err := xmltv.Generate(channels, programs)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	xml := string(data)
	if !strings.Contains(xml, `system="xmltv_ns"`) {
		t.Fatal("expected episode-num with xmltv_ns system")
	}
	if !strings.Contains(xml, "1.79.") {
		t.Fatal("expected season 2, episode 80 as 1.79.")
	}
}

func TestGenerateEmpty(t *testing.T) {
	data, err := xmltv.Generate(nil, nil)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	xml := string(data)
	if !strings.Contains(xml, `<tv`) {
		t.Fatal("expected <tv> element even with empty data")
	}
}
