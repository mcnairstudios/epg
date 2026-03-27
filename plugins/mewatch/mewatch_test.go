package mewatch_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/mewatch"
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
		ID:      "mewatch-ch5",
		XmltvID: "Channel5.sg",
		SiteID:  "97098",
		Lang:    "en",
	}

	programs, err := mewatch.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]
	if p.Title != "Open Homes S3 - EP 2" {
		t.Fatalf("unexpected title: %q", p.Title)
	}

	if p.Source != "mewatch.sg" {
		t.Fatalf("expected source 'mewatch.sg', got %q", p.Source)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := mewatch.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
