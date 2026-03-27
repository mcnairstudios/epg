package horizontv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/horizontv"
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
		ID:      "horizontv-10024",
		XmltvID: "AMCCzechRepublic.cz",
		SiteID:  "10024",
		Lang:    "sk",
	}

	programs, err := horizontv.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Avengement" {
		t.Fatalf("expected title 'Avengement', got %q", p.Title)
	}
	if p.Start.Format("2006-01-02T15:04:05.000Z") != "2023-02-06T21:35:00.000Z" {
		t.Fatalf("unexpected start: %v", p.Start)
	}
	if p.Stop.Format("2006-01-02T15:04:05.000Z") != "2023-02-06T23:05:00.000Z" {
		t.Fatalf("unexpected stop: %v", p.Stop)
	}
}

func TestParseWithDetails(t *testing.T) {
	seg1, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}
	seg2, err := os.ReadFile(testdataPath("content_1.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}
	seg3, err := os.ReadFile(testdataPath("content_2.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}
	seg4, err := os.ReadFile(testdataPath("content_3.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	detail1, err := os.ReadFile(testdataPath("content_listings_1.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}
	detail2, err := os.ReadFile(testdataPath("content_listings_2.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}
	detail3, err := os.ReadFile(testdataPath("content_listings_3.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}
	detail4, err := os.ReadFile(testdataPath("content_listings_4.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "horizontv-10024",
		XmltvID: "AMCCzechRepublic.cz",
		SiteID:  "10024",
		Lang:    "sk",
	}

	details := map[string][]byte{
		"crid:~~2F~~2Fport.cs~~2F122941980,imi:7ca159c917344e0dd3fbe1cd8db5ff8043d96a78": detail1,
		"crid:~~2F~~2Fport.cs~~2F248281986,imi:e85129f9d1e211406a521df7a36f22237c22651b": detail2,
		"crid:~~2F~~2Fport.cs~~2F1379541,imi:5f806a2a0bc13e9745e14907a27116c60ea2c6ad":   detail3,
		"crid:~~2F~~2Fport.cs~~2F71927954,imi:f1b4b0285b72cf44cba74e1c62322a4c682385c7":  detail4,
	}

	programs, err := horizontv.ParseWithDetails(
		[][]byte{seg1, seg2, seg3, seg4},
		details,
		channel,
	)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 4 {
		t.Fatalf("expected 4 programs, got %d", len(programs))
	}

	if programs[0].Title != "Avengement" {
		t.Fatalf("expected title 'Avengement', got %q", programs[0].Title)
	}
	if programs[0].Start.Format("2006-01-02T15:04:05.000Z") != "2023-02-06T21:35:00.000Z" {
		t.Fatalf("unexpected start: %v", programs[0].Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}
	channel := domain.Channel{SiteID: "10024"}
	programs, err := horizontv.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	channel := domain.Channel{SiteID: "10024"}
	programs, err := horizontv.Parse([]byte(`not json`), channel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
