package nzxmltv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/nzxmltv"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.xml"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "nzxmltv-1",
		XmltvID: "TVNZ1.nz",
		SiteID:  "xmltv/guide#1",
		Lang:    "en",
	}
	date := time.Date(2023, 11, 21, 0, 0, 0, 0, time.UTC)

	programs, err := nzxmltv.Parse(data, channel, "1", date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Sunday" {
		t.Fatalf("expected title 'Sunday', got %q", p.Title)
	}

	if p.Description != "On Sunday, an unmissable show with stories about divorce, weight loss, and the incomprehensible devastation of Gaza." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	expectedStart := time.Date(2023, 11, 21, 10, 30, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 11, 21, 11, 25, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Season != 2023 {
		t.Fatalf("expected season 2023, got %d", p.Season)
	}

	if p.Episode != 37 {
		t.Fatalf("expected episode 37, got %d", p.Episode)
	}

	if p.Image != "https://www.thetvdb.com/banners/posters/5dbebff2986f2.jpg" {
		t.Fatalf("expected icon URL, got %q", p.Image)
	}

	if p.Channel != "TVNZ1.nz" {
		t.Fatalf("expected channel 'TVNZ1.nz', got %q", p.Channel)
	}

	if p.Source != "nzxmltv.com" {
		t.Fatalf("expected source 'nzxmltv.com', got %q", p.Source)
	}
}

func TestParseFiltersByDate(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.xml"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "nzxmltv-1",
		XmltvID: "TVNZ1.nz",
		SiteID:  "xmltv/guide#1",
		Lang:    "en",
	}
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	programs, err := nzxmltv.Parse(data, channel, "1", date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 0 {
		t.Fatalf("expected 0 programs for wrong date, got %d", len(programs))
	}
}

func TestParseFiltersByChannel(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.xml"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "nzxmltv-999",
		XmltvID: "Unknown.nz",
		SiteID:  "xmltv/guide#999",
		Lang:    "en",
	}
	date := time.Date(2023, 11, 21, 0, 0, 0, 0, time.UTC)

	programs, err := nzxmltv.Parse(data, channel, "999", date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 0 {
		t.Fatalf("expected 0 programs for unknown channel, got %d", len(programs))
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := nzxmltv.Parse(
		[]byte{},
		domain.Channel{XmltvID: "test"},
		"ch1",
		time.Now(),
	)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidXML(t *testing.T) {
	_, err := nzxmltv.Parse(
		[]byte(`not xml`),
		domain.Channel{},
		"ch1",
		time.Now(),
	)
	if err == nil {
		t.Fatal("expected error on invalid XML")
	}
}

func TestParseMultiplePrograms(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.xml"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "nzxmltv-1",
		XmltvID: "TVNZ1.nz",
		SiteID:  "xmltv/guide#1",
		Lang:    "en",
	}
	date := time.Date(2023, 11, 21, 0, 0, 0, 0, time.UTC)

	programs, err := nzxmltv.Parse(data, channel, "1", date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	if programs[1].Title != "Rich House, Poor House" {
		t.Fatalf("expected second title 'Rich House, Poor House', got %q", programs[1].Title)
	}

	if programs[1].Season != 8 {
		t.Fatalf("expected season 8, got %d", programs[1].Season)
	}

	if programs[1].Episode != 6 {
		t.Fatalf("expected episode 6, got %d", programs[1].Episode)
	}
}
