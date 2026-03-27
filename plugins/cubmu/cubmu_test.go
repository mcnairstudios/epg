package cubmu_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/cubmu"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContentID(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "cubmu-transtv",
		XmltvID: "TransTV.id",
		SiteID:  "4028c68574537fcd0174be43042758d8",
		Lang:    "id",
	}

	programs, err := cubmu.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "CNN Tech News" {
		t.Fatalf("expected title 'CNN Tech News', got %q", p.Title)
	}

	expectedStart := time.Date(2023, 11, 4, 18, 30, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 11, 4, 19, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	expectedDesc := "CNN Indonesia Tech News adalah berita teknologi yang membawa pemirsa ke dunia teknologi yang penuh dengan informasi, pendidikan, hiburan sampai informasi kesehatan terkini."
	if p.Description != expectedDesc {
		t.Fatalf("unexpected description: %q", p.Description)
	}
}

func TestParseContentEN(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "cubmu-transtv",
		XmltvID: "TransTV.id",
		SiteID:  "4028c68574537fcd0174be43042758d8",
		Lang:    "en",
	}

	programs, err := cubmu.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	expectedDesc := "CNN Indonesia Tech News is tech news brings viewers into the world of technology that provides information, education, entertainment to the latest health information."
	if programs[0].Description != expectedDesc {
		t.Fatalf("unexpected description: %q", programs[0].Description)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := cubmu.Parse([]byte(""), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
