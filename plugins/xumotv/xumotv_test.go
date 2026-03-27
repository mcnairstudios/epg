package xumotv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/xumotv"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_0.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "xumo-99991247",
		XmltvID: "NBCNewsNow.us",
		SiteID:  "0#99991247",
		Lang:    "en",
	}

	programs, err := xumotv.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Dateline" {
		t.Fatalf("expected title 'Dateline', got %q", p.Title)
	}

	if p.SubTitle != "The Disappearance of Laci Peterson" {
		t.Fatalf("expected subtitle 'The Disappearance of Laci Peterson', got %q", p.SubTitle)
	}

	expectedDesc := "After following Laci Peterson's case for more than 15 years, the show delivers a comprehensive report with rarely seen interrogation video, new insight from prosecutors, and surprising details from Amber Frey, who helped uncover the truth."
	if p.Description != expectedDesc {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	expectedStart := time.Date(2022, 11, 5, 23, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 11, 6, 1, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "NBCNewsNow.us" {
		t.Fatalf("expected channel 'NBCNewsNow.us', got %q", p.Channel)
	}

	if p.Source != "xumo.tv" {
		t.Fatalf("expected source 'xumo.tv', got %q", p.Source)
	}
}

func TestParseItemsFromMultiplePages(t *testing.T) {
	channel := domain.Channel{
		SiteID: "0#99991247",
	}

	data0, err := os.ReadFile(testdataPath("content_0.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	items0 := xumotv.ParseItems(data0, channel)
	if len(items0) == 0 {
		t.Fatal("expected items from page 0")
	}

	data1, err := os.ReadFile(testdataPath("content_1.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	items1 := xumotv.ParseItems(data1, channel)
	if len(items1) == 0 {
		t.Fatal("expected items from page 1")
	}

	if items1[0].Title != "Incredible World" {
		t.Fatalf("expected first item from page 1 to be 'Incredible World', got %q", items1[0].Title)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no-content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "0#99991247",
		XmltvID: "NBCNewsNow.us",
	}

	programs, err := xumotv.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	channel := domain.Channel{SiteID: "0#99991247"}
	programs, err := xumotv.Parse([]byte(`not json`), channel)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
