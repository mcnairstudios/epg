package telenet_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/telenet"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseSegment(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_0000.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	events, err := telenet.ParseSegment(data, "outtv")
	if err != nil {
		t.Fatalf("parse segment failed: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	if events[0].Title != "Queer as Folk USA" {
		t.Fatalf("expected title 'Queer as Folk USA', got %q", events[0].Title)
	}

	expectedStart := time.Date(2022, 10, 29, 23, 56, 0, 0, time.UTC)
	if !time.Unix(events[0].StartTime, 0).UTC().Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, time.Unix(events[0].StartTime, 0).UTC())
	}

	expectedStop := time.Date(2022, 10, 30, 1, 44, 0, 0, time.UTC)
	if !time.Unix(events[0].EndTime, 0).UTC().Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, time.Unix(events[0].EndTime, 0).UTC())
	}
}

func TestParseSegmentMissingChannel(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_0000.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	events, err := telenet.ParseSegment(data, "nonexistent")
	if err != nil {
		t.Fatalf("parse segment failed: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestParseSegmentEmpty(t *testing.T) {
	events, err := telenet.ParseSegment([]byte{}, "outtv")
	if err != nil {
		t.Fatalf("parse segment failed: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestParseSegmentInvalidJSON(t *testing.T) {
	_, err := telenet.ParseSegment([]byte("not json"), "outtv")
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseDetail(t *testing.T) {
	data, err := os.ReadFile(testdataPath("program.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	detail, err := telenet.ParseDetail(data)
	if err != nil {
		t.Fatalf("parse detail failed: %v", err)
	}

	if detail == nil {
		t.Fatal("expected non-nil detail")
	}

	if detail.LongDescription == "" {
		t.Fatal("expected non-empty long description")
	}

	if detail.SeasonNumber != 5 {
		t.Fatalf("expected season 5, got %d", detail.SeasonNumber)
	}

	if detail.EpisodeNumber != 8 {
		t.Fatalf("expected episode 8, got %d", detail.EpisodeNumber)
	}

	if len(detail.Genres) != 2 {
		t.Fatalf("expected 2 genres, got %d", len(detail.Genres))
	}

	if detail.Genres[0] != "Dramaserie" {
		t.Fatalf("expected genre 'Dramaserie', got %q", detail.Genres[0])
	}

	if len(detail.Actors) != 8 {
		t.Fatalf("expected 8 actors, got %d", len(detail.Actors))
	}
}

func TestParseDetailEmpty(t *testing.T) {
	detail, err := telenet.ParseDetail([]byte{})
	if err != nil {
		t.Fatalf("parse detail failed: %v", err)
	}
	if detail != nil {
		t.Fatal("expected nil detail for empty data")
	}
}

func TestParseDetailInvalidJSON(t *testing.T) {
	_, err := telenet.ParseDetail([]byte("not json"))
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseSegmentAllSlots(t *testing.T) {
	slots := []struct {
		file     string
		expected int
	}{
		{"content_0000.json", 3},
		{"content_0600.json", 4},
		{"content_1200.json", 5},
		{"content_1800.json", 10},
	}

	for _, slot := range slots {
		data, err := os.ReadFile(testdataPath(slot.file))
		if err != nil {
			t.Fatalf("failed to read %s: %v", slot.file, err)
		}

		events, err := telenet.ParseSegment(data, "outtv")
		if err != nil {
			t.Fatalf("parse segment %s failed: %v", slot.file, err)
		}

		if len(events) != slot.expected {
			t.Fatalf("%s: expected %d events, got %d", slot.file, slot.expected, len(events))
		}
	}
}

func TestIntegrationProgramBuild(t *testing.T) {
	segData, err := os.ReadFile(testdataPath("content_0000.json"))
	if err != nil {
		t.Fatalf("failed to read segment data: %v", err)
	}

	detailData, err := os.ReadFile(testdataPath("program.json"))
	if err != nil {
		t.Fatalf("failed to read detail data: %v", err)
	}

	events, err := telenet.ParseSegment(segData, "outtv")
	if err != nil {
		t.Fatalf("parse segment failed: %v", err)
	}

	detail, err := telenet.ParseDetail(detailData)
	if err != nil {
		t.Fatalf("parse detail failed: %v", err)
	}

	channel := domain.Channel{
		ID:      "telenet.tv-outtv",
		XmltvID: "OutTV.nl",
		SiteID:  "outtv",
		Lang:    "nl",
	}

	ev := events[0]
	prog := domain.Program{
		Channel:     channel.XmltvID,
		Source:      "telenet.tv",
		Title:       ev.Title,
		Start:       time.Unix(ev.StartTime, 0).UTC(),
		Stop:        time.Unix(ev.EndTime, 0).UTC(),
		Lang:        channel.Lang,
		Description: detail.LongDescription,
		Season:      detail.SeasonNumber,
		Episode:     detail.EpisodeNumber,
		Category:    "Dramaserie, LHBTI",
	}

	if prog.Title != "Queer as Folk USA" {
		t.Fatalf("expected title 'Queer as Folk USA', got %q", prog.Title)
	}

	if prog.Channel != "OutTV.nl" {
		t.Fatalf("expected channel 'OutTV.nl', got %q", prog.Channel)
	}

	if prog.Source != "telenet.tv" {
		t.Fatalf("expected source 'telenet.tv', got %q", prog.Source)
	}

	expectedStart := time.Date(2022, 10, 29, 23, 56, 0, 0, time.UTC)
	if !prog.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, prog.Start)
	}

	expectedStop := time.Date(2022, 10, 30, 1, 44, 0, 0, time.UTC)
	if !prog.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, prog.Stop)
	}

	if prog.Season != 5 {
		t.Fatalf("expected season 5, got %d", prog.Season)
	}

	if prog.Episode != 8 {
		t.Fatalf("expected episode 8, got %d", prog.Episode)
	}
}
