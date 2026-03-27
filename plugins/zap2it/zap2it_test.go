package zap2it_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/zap2it"
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
		ID:      "zap2it.com-X/USA-NY31695-DEFAULT/NY31695/USA/13302/49141",
		XmltvID: "SpectrumNews1.us",
		SiteID:  "X/USA-NY31695-DEFAULT/NY31695/USA/13302/49141",
		Lang:    "en",
	}

	programs, err := zap2it.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Your Afternoon on Spectrum News 1 - Central NY" {
		t.Fatalf("expected title 'Your Afternoon on Spectrum News 1 - Central NY', got %q", p.Title)
	}

	if p.SubTitle != "Episode 42" {
		t.Fatalf("expected subtitle 'Episode 42', got %q", p.SubTitle)
	}

	if p.Description != "Local news coverage" {
		t.Fatalf("expected description 'Local news coverage', got %q", p.Description)
	}

	if p.Category != "News" {
		t.Fatalf("expected category 'News', got %q", p.Category)
	}

	expectedStart := time.Unix(1738868400, 0).UTC()
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Unix(1738872000, 0).UTC()
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Season != 3 {
		t.Fatalf("expected season 3, got %d", p.Season)
	}

	if p.Episode != 42 {
		t.Fatalf("expected episode 42, got %d", p.Episode)
	}

	if p.Image != "https://zap2it.tmsimg.com/assets/p12345_i_h9_ab.jpg" {
		t.Fatalf("expected image URL with thumbnail, got %q", p.Image)
	}

	if p.Channel != "SpectrumNews1.us" {
		t.Fatalf("expected channel 'SpectrumNews1.us', got %q", p.Channel)
	}

	if p.Source != "zap2it.com" {
		t.Fatalf("expected source 'zap2it.com', got %q", p.Source)
	}

	if p.Lang != "en" {
		t.Fatalf("expected lang 'en', got %q", p.Lang)
	}
}

func TestParseSecondProgram(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		XmltvID: "SpectrumNews1.us",
		Lang:    "en",
	}

	programs, err := zap2it.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	p := programs[1]

	if p.Title != "Evening Report" {
		t.Fatalf("expected title 'Evening Report', got %q", p.Title)
	}

	if p.SubTitle != "" {
		t.Fatalf("expected empty subtitle, got %q", p.SubTitle)
	}

	if p.Description != "" {
		t.Fatalf("expected empty description, got %q", p.Description)
	}

	if p.Category != "" {
		t.Fatalf("expected empty category, got %q", p.Category)
	}

	if p.Image != "" {
		t.Fatalf("expected empty image, got %q", p.Image)
	}

	if p.Season != 0 {
		t.Fatalf("expected season 0, got %d", p.Season)
	}

	if p.Episode != 0 {
		t.Fatalf("expected episode 0, got %d", p.Episode)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := zap2it.Parse([]byte(`{}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := zap2it.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
