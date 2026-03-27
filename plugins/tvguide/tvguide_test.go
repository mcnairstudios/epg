package tvguide_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/tvguide"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseResponse(t *testing.T) {
	content, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programData, err := os.ReadFile(testdataPath("program.json"))
	if err != nil {
		t.Fatalf("failed to read program data: %v", err)
	}

	var detailsResp struct {
		Data struct {
			Item *tvguide.ItemDetails `json:"item"`
		} `json:"data"`
	}
	if err := json.Unmarshal(programData, &detailsResp); err != nil {
		t.Fatalf("failed to unmarshal program: %v", err)
	}

	detailsMap := map[string]*tvguide.ItemDetails{}
	if detailsResp.Data.Item != nil {
		detailsMap["https://backend.tvguide.com/tvschedules/tvguide/programdetails/9000058285/web"] = detailsResp.Data.Item
	}

	channel := domain.Channel{
		SiteID:  "9200004683",
		XmltvID: "NationalGeographicWild.us@East",
	}

	fetcher := func(url string) *tvguide.ItemDetails {
		return detailsMap[url]
	}

	programs, err := tvguide.Parse(content, channel, fetcher)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Secrets of the Zoo: North Carolina" {
		t.Fatalf("expected title 'Secrets of the Zoo: North Carolina', got %q", p.Title)
	}

	if p.SubTitle != "Chimp Off the Old Block" {
		t.Fatalf("expected subtitle 'Chimp Off the Old Block', got %q", p.SubTitle)
	}

	expectedDesc := "Chimps living at the North Carolina Zoo, a zoo located in the center of North Carolina that serves as the world's largest natural habitat zoo, as well as one of two state-supported zoos, are cared for"
	if p.Description != expectedDesc {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	if p.Season != 1 {
		t.Fatalf("expected season 1, got %d", p.Season)
	}

	if p.Episode != 1 {
		t.Fatalf("expected episode 1, got %d", p.Episode)
	}

	if p.Category != "Reality" {
		t.Fatalf("expected category 'Reality', got %q", p.Category)
	}

	expectedStart := "2025-07-29T00:00:00Z"
	if p.Start.UTC().Format("2006-01-02T15:04:05Z") != expectedStart {
		t.Fatalf("expected start %s, got %s", expectedStart, p.Start.UTC().Format("2006-01-02T15:04:05Z"))
	}

	expectedStop := "2025-07-29T01:00:00Z"
	if p.Stop.UTC().Format("2006-01-02T15:04:05Z") != expectedStop {
		t.Fatalf("expected stop %s, got %s", expectedStop, p.Stop.UTC().Format("2006-01-02T15:04:05Z"))
	}
}

func TestParseEmptyContent(t *testing.T) {
	content, err := os.ReadFile(testdataPath("no-content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "9200004683",
		XmltvID: "NationalGeographicWild.us@East",
	}

	programs, err := tvguide.Parse(content, channel, nil)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
