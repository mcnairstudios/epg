package whaletvplus_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/whaletvplus"
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
		ID:      "watch.whaletvplus.com-972712064533996134",
		XmltvID: "123GO.whaletvplus.com",
		SiteID:  "972712064533996134",
		Lang:    "en",
	}

	programs, err := whaletvplus.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "123GO!" {
		t.Fatalf("expected title '123GO!', got %q", p.Title)
	}

	expectedStart := time.Date(2026, 1, 6, 6, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2026, 1, 6, 6, 25, 30, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "123GO.whaletvplus.com" {
		t.Fatalf("expected channel '123GO.whaletvplus.com', got %q", p.Channel)
	}

	if p.Source != "watch.whaletvplus.com" {
		t.Fatalf("expected source 'watch.whaletvplus.com', got %q", p.Source)
	}

	if p.Lang != "en" {
		t.Fatalf("expected lang 'en', got %q", p.Lang)
	}
}

func TestParseMultipleChannels(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "972712064533996134",
		XmltvID: "Test.Channel",
		Lang:    "en",
	}

	programs, err := whaletvplus.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 58 {
		t.Fatalf("expected 58 programs from first channel data block, got %d", len(programs))
	}
}

func TestParseEmptyData(t *testing.T) {
	programs, err := whaletvplus.Parse([]byte(`{"data":[]}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := whaletvplus.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseNoData(t *testing.T) {
	programs, err := whaletvplus.Parse([]byte(`{}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseSingleChannelResponse(t *testing.T) {
	data := []byte(`{"data":[{"chlId":"123","chlNm":"Test","ptList":[{"prgStm":"1767679200000","prgEtm":"1767680730000","prgTitle":"Show A","prgchId":"1"},{"prgStm":"1767680730000","prgEtm":"1767682200000","prgTitle":"Show B","prgchId":"2"}]}]}`)

	channel := domain.Channel{
		SiteID:  "123",
		XmltvID: "Test.whaletvplus.com",
		Lang:    "de",
	}

	programs, err := whaletvplus.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	if programs[0].Title != "Show A" {
		t.Fatalf("expected title 'Show A', got %q", programs[0].Title)
	}

	if programs[1].Title != "Show B" {
		t.Fatalf("expected title 'Show B', got %q", programs[1].Title)
	}

	if programs[0].Lang != "de" {
		t.Fatalf("expected lang 'de', got %q", programs[0].Lang)
	}
}
