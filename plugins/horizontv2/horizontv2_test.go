package horizontv2_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/horizontv2"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParse(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "10024",
		XmltvID: "AMCCzechRepublic.cz",
		Lang:    "sk",
	}

	programs, err := horizontv2.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Avengement" {
		t.Fatalf("expected title 'Avengement', got %q", p.Title)
	}

	expectedStart := time.Date(2023, 2, 6, 21, 35, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 2, 6, 23, 5, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
}

func TestParseWithDetails(t *testing.T) {
	segments := make([][]byte, 4)
	for i, name := range []string{"content.json", "content_1.json", "content_2.json", "content_3.json"} {
		data, err := os.ReadFile(testdataPath(name))
		if err != nil {
			t.Fatalf("failed to read %s: %v", name, err)
		}
		segments[i] = data
	}

	details := map[string][]byte{}
	detailFiles := map[string]string{
		"crid:~~2F~~2Fport.cs~~2F122941980,imi:7ca159c917344e0dd3fbe1cd8db5ff8043d96a78":  "content_listings_1.json",
		"crid:~~2F~~2Fport.cs~~2F248281986,imi:e85129f9d1e211406a521df7a36f22237c22651b":  "content_listings_2.json",
		"crid:~~2F~~2Fport.cs~~2F1379541,imi:5f806a2a0bc13e9745e14907a27116c60ea2c6ad":    "content_listings_3.json",
		"crid:~~2F~~2Fport.cs~~2F71927954,imi:f1b4b0285b72cf44cba74e1c62322a4c682385c7":  "content_listings_4.json",
	}
	for id, filename := range detailFiles {
		data, err := os.ReadFile(testdataPath(filename))
		if err != nil {
			t.Fatalf("failed to read %s: %v", filename, err)
		}
		details[id] = data
	}

	channel := domain.Channel{
		SiteID:  "10024",
		XmltvID: "AMCCzechRepublic.cz",
		Lang:    "sk",
	}

	programs, err := horizontv2.ParseWithDetails(segments, details, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 4 {
		t.Fatalf("expected 4 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Avengement" {
		t.Fatalf("expected title 'Avengement', got %q", p.Title)
	}
	if p.Description == "" {
		t.Fatal("expected description to be set")
	}

	expectedStart := time.Date(2023, 2, 6, 21, 35, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := horizontv2.Parse(data, domain.Channel{SiteID: "10024"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
