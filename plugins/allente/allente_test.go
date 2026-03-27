package allente_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/allente"
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
		SiteID:  "se#0148",
		XmltvID: "SVT1.se",
		Lang:    "se",
	}

	programs, err := allente.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Hemmagympa med Sofia" {
		t.Fatalf("expected title 'Hemmagympa med Sofia', got %q", p.Title)
	}

	if p.Description != "Svenskt träningsprogram från 2021. Styrka. Sofia Åhman leder SVT:s hemmagympapass. Denna gång fokuserar vi på styrka." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	if p.Season != 4 {
		t.Fatalf("expected season 4, got %d", p.Season)
	}

	if p.Episode != 1 {
		t.Fatalf("expected episode 1, got %d", p.Episode)
	}

	if p.Category != "other" {
		t.Fatalf("expected category 'other', got %q", p.Category)
	}

	expectedStart := "2022-08-22T07:10:00Z"
	if p.Start.Format("2006-01-02T15:04:05Z") != expectedStart {
		t.Fatalf("expected start %s, got %v", expectedStart, p.Start)
	}

	expectedStop := "2022-08-22T07:30:00Z"
	if p.Stop.Format("2006-01-02T15:04:05Z") != expectedStop {
		t.Fatalf("expected stop %s, got %v", expectedStop, p.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "se#0148",
		XmltvID: "SVT1.se",
		Lang:    "se",
	}

	programs, err := allente.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := allente.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
