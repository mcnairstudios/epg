package tvpassport_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvpassport"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "tvpassport-5463",
		XmltvID: "YTATV.us",
		SiteID:  "youtoo-america-network/5463",
		Lang:    "en",
	}

	programs, err := tvpassport.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Charlie Moore: No Offense" {
		t.Fatalf("expected title 'Charlie Moore: No Offense', got %q", p.Title)
	}

	if p.SubTitle != "Under the Influencer" {
		t.Fatalf("expected subtitle 'Under the Influencer', got %q", p.SubTitle)
	}

	expectedStart := time.Date(2022, 10, 4, 10, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 10, 4, 10, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Category != "Sports, Outdoors" {
		t.Fatalf("expected category 'Sports, Outdoors', got %q", p.Category)
	}

	if p.Image != "https://cdn.tvpassport.com/image/show/960x540/69103.jpg" {
		t.Fatalf("expected image 'https://cdn.tvpassport.com/image/show/960x540/69103.jpg', got %q", p.Image)
	}

	if p.Description != "Celebrity interviews while fishing in various locations throughout the United States." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	p2 := programs[2]
	if p2.Title != "The Mark of Zorro" {
		t.Fatalf("expected title 'The Mark of Zorro', got %q", p2.Title)
	}

	expectedStart2 := time.Date(2022, 10, 4, 11, 0, 0, 0, time.UTC)
	if !p2.Start.Equal(expectedStart2) {
		t.Fatalf("expected start %v, got %v", expectedStart2, p2.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := tvpassport.Parse([]byte(""), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidHTML(t *testing.T) {
	programs, err := tvpassport.Parse([]byte("<html><body></body></html>"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
