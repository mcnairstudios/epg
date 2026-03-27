package iltalehti_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/iltalehti"
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
		ID:      "iltalehti-yle1",
		XmltvID: "YleTV1.fi",
		SiteID:  "d63736b5-7f72-45d4-8f9e-6505f1bac855#391",
		Lang:    "fi",
	}

	programs, err := iltalehti.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	found := false
	for _, p := range programs {
		if p.Title == "Ykkösaamu" {
			found = true
			expectedStart := time.Date(2025, 12, 14, 5, 0, 0, 0, time.UTC)
			if !p.Start.Equal(expectedStart) {
				t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
			}

			expectedStop := time.Date(2025, 12, 14, 5, 40, 0, 0, time.UTC)
			if !p.Stop.Equal(expectedStop) {
				t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
			}

			if p.Image == "" {
				t.Fatal("expected image to be set")
			}
			break
		}
	}

	if !found {
		t.Fatal("expected to find program 'Ykkösaamu'")
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no-content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID: "d63736b5-7f72-45d4-8f9e-6505f1bac855#391",
	}

	programs, err := iltalehti.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := iltalehti.Parse([]byte("not json"), domain.Channel{SiteID: "x#y"})
	if err != nil {
		t.Fatalf("parse should not error, got: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
