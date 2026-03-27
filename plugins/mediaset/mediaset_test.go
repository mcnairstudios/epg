package mediaset_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/mediaset"
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
		ID:      "mediaset-LB",
		XmltvID: "20.it",
		SiteID:  "LB",
		Lang:    "it",
	}

	programs, err := mediaset.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[3]

	if p.Title != "Chicago Fire" {
		t.Fatalf("expected title 'Chicago Fire', got %q", p.Title)
	}

	if p.SubTitle != "Ep. 22 - Io non ti lascio" {
		t.Fatalf("expected subtitle 'Ep. 22 - Io non ti lascio', got %q", p.SubTitle)
	}

	if p.Description != "Severide e Kidd continuano a indagare su un vecchio caso doloso di Benny. Notizie inaspettate portano Brett a meditare su una grande decisione." {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	if p.Category != "Intrattenimento" {
		t.Fatalf("expected category 'Intrattenimento', got %q", p.Category)
	}

	if p.Season != 7 {
		t.Fatalf("expected season 7, got %d", p.Season)
	}

	if p.Episode != 22 {
		t.Fatalf("expected episode 22, got %d", p.Episode)
	}

	expectedStart := time.Date(2024, 1, 20, 2, 14, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2024, 1, 20, 2, 54, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image != "https://static2.mediasetplay.mediaset.it/Mediaset_Italia_Production_-_Main/F309370301002204/media/0/0/1ef76b73-3173-43bd-9c16-73986a0ec131/46896726-11e7-4438-b947-d2ae53f58c0b.jpg" {
		t.Fatalf("unexpected image: %q", p.Image)
	}

	if p.Channel != "20.it" {
		t.Fatalf("expected channel '20.it', got %q", p.Channel)
	}

	if p.Source != "mediasetinfinity.mediaset.it" {
		t.Fatalf("expected source 'mediasetinfinity.mediaset.it', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := mediaset.Parse([]byte(`[]`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := mediaset.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNoEntries(t *testing.T) {
	programs, err := mediaset.Parse([]byte(`{"response":{"entries":[]}}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
