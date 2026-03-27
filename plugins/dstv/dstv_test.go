package dstv_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/dstv"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

type programDetails struct {
	Synopsis     string   `json:"Synopsis"`
	ThumbnailURI string   `json:"ThumbnailUri"`
	SubGenres    []string `json:"SubGenres"`
}

func TestParseResponseZA(t *testing.T) {
	content, err := os.ReadFile(testdataPath("content_zaf.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	detailsBytes, err := os.ReadFile(testdataPath("program_zaf.json"))
	if err != nil {
		t.Fatalf("failed to read program details: %v", err)
	}

	var details programDetails
	if err := json.Unmarshal(detailsBytes, &details); err != nil {
		t.Fatalf("failed to unmarshal details: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "zaf#201",
		XmltvID: "SuperSportGrandstand.za",
	}

	programs, err := dstv.Parse(content, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	if len(programs) < 2 {
		t.Fatal("expected at least two programs")
	}

	p := programs[1]

	if p.Title != "UFC FN HL: Nzechukwu v Cutelaba" {
		t.Fatalf("expected title 'UFC FN HL: Nzechukwu v Cutelaba', got %q", p.Title)
	}

	expectedStart := "2022-11-21T23:00:00Z"
	if p.Start.UTC().Format("2006-01-02T15:04:05Z") != expectedStart {
		t.Fatalf("expected start %s, got %s", expectedStart, p.Start.UTC().Format("2006-01-02T15:04:05Z"))
	}

	expectedStop := "2022-11-22T00:00:00Z"
	if p.Stop.UTC().Format("2006-01-02T15:04:05Z") != expectedStop {
		t.Fatalf("expected stop %s, got %s", expectedStop, p.Stop.UTC().Format("2006-01-02T15:04:05Z"))
	}
}

func TestParseResponseNG(t *testing.T) {
	content, err := os.ReadFile(testdataPath("content_nga.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "nga#201",
		XmltvID: "SuperSportGrandstand.za",
	}

	programs, err := dstv.Parse(content, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "UFC FN HL: Nzechukwu v Cutelaba" {
		t.Fatalf("expected title 'UFC FN HL: Nzechukwu v Cutelaba', got %q", p.Title)
	}

	expectedStart := "2022-11-21T23:00:00Z"
	if p.Start.UTC().Format("2006-01-02T15:04:05Z") != expectedStart {
		t.Fatalf("expected start %s, got %s", expectedStart, p.Start.UTC().Format("2006-01-02T15:04:05Z"))
	}

	expectedStop := "2022-11-22T00:00:00Z"
	if p.Stop.UTC().Format("2006-01-02T15:04:05Z") != expectedStop {
		t.Fatalf("expected stop %s, got %s", expectedStop, p.Stop.UTC().Format("2006-01-02T15:04:05Z"))
	}
}

func TestParseWithDetailsZA(t *testing.T) {
	content, err := os.ReadFile(testdataPath("content_zaf.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	detailsBytes, err := os.ReadFile(testdataPath("program_zaf.json"))
	if err != nil {
		t.Fatalf("failed to read program details: %v", err)
	}

	var rawDetails struct {
		ID           string   `json:"Id"`
		Synopsis     string   `json:"Synopsis"`
		ThumbnailURI string   `json:"ThumbnailUri"`
		SubGenres    []string `json:"SubGenres"`
	}
	if err := json.Unmarshal(detailsBytes, &rawDetails); err != nil {
		t.Fatalf("failed to unmarshal details: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "zaf#201",
		XmltvID: "SuperSportGrandstand.za",
	}

	detailsMap := map[string]*dstv.ProgramDetails{
		rawDetails.ID: {
			Synopsis:     rawDetails.Synopsis,
			ThumbnailURI: rawDetails.ThumbnailURI,
			SubGenres:    rawDetails.SubGenres,
		},
	}

	programs, err := dstv.ParseWithDetails(content, channel, detailsMap)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) < 2 {
		t.Fatal("expected at least two programs")
	}

	p := programs[1]

	if p.Description != "'UFC Fight Night Highlights - Heavyweight Bout: Kennedy Nzechukwu vs Ion Cutelaba'. From The UFC APEX Center - Las Vegas, USA." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	if p.Image != "https://03mcdecdnimagerepository.blob.core.windows.net/epguideimage/img/271546_UFC Fight Night.png" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	if p.Category != "All Sport, Mixed Martial Arts" {
		t.Fatalf("unexpected category: %q", p.Category)
	}
}

func TestParseEmptyContent(t *testing.T) {
	channel := domain.Channel{
		SiteID:  "zaf#201",
		XmltvID: "SuperSportGrandstand.za",
	}

	programs, err := dstv.Parse([]byte(`{"Total":0,"Channels":[]}`), channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
