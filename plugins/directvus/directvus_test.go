package directvus_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/directvus"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func loadDetails(t *testing.T) map[string]*directvus.ProgramDetail {
	t.Helper()
	details := make(map[string]*directvus.ProgramDetail)

	for _, entry := range []struct {
		id   string
		file string
	}{
		{"MV001173520000", "program1.json"},
		{"EP002298270445", "program2.json"},
	} {
		data, err := os.ReadFile(testdataPath(entry.file))
		if err != nil {
			t.Fatalf("failed to read %s: %v", entry.file, err)
		}
		var resp struct {
			ProgramDetail *directvus.ProgramDetail `json:"programDetail"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			t.Fatalf("failed to parse %s: %v", entry.file, err)
		}
		details[entry.id] = resp.ProgramDetail
	}

	return details
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "249#249",
		XmltvID: "ComedyCentralEast.us",
		Lang:    "en",
	}

	details := loadDetails(t)

	programs, err := directvus.Parse(data, channel, details)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	p0 := programs[0]
	if p0.Title != "Men in Black II" {
		t.Fatalf("expected title 'Men in Black II', got %q", p0.Title)
	}
	if p0.Description == "" {
		t.Fatal("expected description to be set")
	}
	expectedStart := time.Date(2023, 1, 14, 23, 0, 0, 0, time.UTC)
	if !p0.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p0.Start)
	}
	expectedStop := time.Date(2023, 1, 15, 1, 0, 0, 0, time.UTC)
	if !p0.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p0.Stop)
	}
	if p0.Image != "https://www.directv.com/db_photos/movies/AllPhotosAPGI/29160/29160_aa.jpg" {
		t.Fatalf("unexpected image %q", p0.Image)
	}
	if p0.Category != "Comedy, Movies Anywhere, Action/Adventure, Science Fiction" {
		t.Fatalf("unexpected category %q", p0.Category)
	}
	if p0.Channel != "ComedyCentralEast.us" {
		t.Fatalf("expected channel 'ComedyCentralEast.us', got %q", p0.Channel)
	}
	if p0.Source != "directv.com" {
		t.Fatalf("expected source 'directv.com', got %q", p0.Source)
	}

	p1 := programs[1]
	if p1.Title != "South Park" {
		t.Fatalf("expected title 'South Park', got %q", p1.Title)
	}
	if p1.SubTitle != "Goth Kids 3: Dawn of the Posers" {
		t.Fatalf("expected subtitle 'Goth Kids 3: Dawn of the Posers', got %q", p1.SubTitle)
	}
	if p1.Season != 17 {
		t.Fatalf("expected season 17, got %d", p1.Season)
	}
	if p1.Episode != 4 {
		t.Fatalf("expected episode 4, got %d", p1.Episode)
	}
	if p1.Description != "The goth kids are sent to a camp for troubled children." {
		t.Fatalf("unexpected description %q", p1.Description)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no-content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "249#249",
		XmltvID: "ComedyCentralEast.us",
	}

	programs, err := directvus.Parse(data, channel, nil)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	channel := domain.Channel{SiteID: "249#249"}
	_, err := directvus.Parse([]byte(`not json`), channel, nil)
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseNoSchedule(t *testing.T) {
	channel := domain.Channel{SiteID: "249#249"}
	programs, err := directvus.Parse([]byte(`{}`), channel, nil)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
