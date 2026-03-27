package playeree_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/playeree"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func loadItems(t *testing.T, files ...string) []playeree.ScheduleItem {
	t.Helper()
	var all []playeree.ScheduleItem
	for _, f := range files {
		data, err := os.ReadFile(testdataPath(f))
		if err != nil {
			t.Fatalf("failed to read %s: %v", f, err)
		}
		items, err := playeree.ParseResponse(data)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", f, err)
		}
		all = append(all, items...)
	}
	return all
}

func TestParseContent(t *testing.T) {
	items := loadItems(t, "data.json", "data1.json")

	channel := domain.Channel{
		SiteID:  "dvb://233a..6d60",
		XmltvID: "HGTV.uk",
		Lang:    "en",
	}

	programs, err := playeree.Parse(items, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Bargain Mansions" {
		t.Fatalf("expected title 'Bargain Mansions', got %q", p.Title)
	}
	if p.Season != 4 {
		t.Fatalf("expected season 4, got %d", p.Season)
	}
	if p.Episode != 1 {
		t.Fatalf("expected episode 1, got %d", p.Episode)
	}
	expectedStart := time.Date(2023, 12, 13, 13, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}
	expectedStop := time.Date(2023, 12, 13, 14, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}
	if p.Channel != "HGTV.uk" {
		t.Fatalf("expected channel 'HGTV.uk', got %q", p.Channel)
	}
	if p.Source != "player.ee.co.uk" {
		t.Fatalf("expected source 'player.ee.co.uk', got %q", p.Source)
	}

	p2 := programs[1]
	if p2.Title != "Flip Or Flop" {
		t.Fatalf("expected title 'Flip Or Flop', got %q", p2.Title)
	}
	if p2.Season != 2 {
		t.Fatalf("expected season 2, got %d", p2.Season)
	}
	if p2.Episode != 2 {
		t.Fatalf("expected episode 2, got %d", p2.Episode)
	}
	expectedStart2 := time.Date(2023, 12, 13, 14, 0, 0, 0, time.UTC)
	if !p2.Start.Equal(expectedStart2) {
		t.Fatalf("expected start %v, got %v", expectedStart2, p2.Start)
	}
	expectedStop2 := time.Date(2023, 12, 13, 14, 30, 0, 0, time.UTC)
	if !p2.Stop.Equal(expectedStop2) {
		t.Fatalf("expected stop %v, got %v", expectedStop2, p2.Stop)
	}
}

func TestParseEmptyItems(t *testing.T) {
	programs, err := playeree.Parse(nil, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseNoSeasonEpisode(t *testing.T) {
	items := []playeree.ScheduleItem{
		{
			Title:              "Test Show",
			Synopsis:           "A show without season info.",
			PublishedStartTime: "2023-12-13T10:00:00Z",
			PublishedDuration:  1800,
		},
	}

	programs, err := playeree.Parse(items, domain.Channel{
		XmltvID: "Test.uk",
		Lang:    "en",
	})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}
	if programs[0].Season != 0 {
		t.Fatalf("expected season 0, got %d", programs[0].Season)
	}
	if programs[0].Episode != 0 {
		t.Fatalf("expected episode 0, got %d", programs[0].Episode)
	}
}

func TestParseResponseInvalidJSON(t *testing.T) {
	_, err := playeree.ParseResponse([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
