package berrymedia_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/berrymedia"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2023, 1, 26, 0, 0, 0, 0, time.UTC)
	channel := domain.Channel{
		SiteID:  "",
		XmltvID: "GTV.kr",
		Lang:    "ko",
	}

	programs, err := berrymedia.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "더트롯쇼" {
		t.Fatalf("expected title '더트롯쇼', got %q", p.Title)
	}

	expectedStart := time.Date(2023, 1, 25, 15, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 1, 25, 16, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Category != "연예/오락" {
		t.Fatalf("expected category '연예/오락', got %q", p.Category)
	}

	if p.Channel != "GTV.kr" {
		t.Fatalf("expected channel 'GTV.kr', got %q", p.Channel)
	}

	if p.Source != "berrymedia.co.kr" {
		t.Fatalf("expected source 'berrymedia.co.kr', got %q", p.Source)
	}

	last := programs[len(programs)-1]
	if last.Title != "나는 자연인이다" {
		t.Fatalf("expected last title '나는 자연인이다', got %q", last.Title)
	}
}

func TestParseNoContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	date := time.Date(2023, 1, 26, 0, 0, 0, 0, time.UTC)
	programs, err := berrymedia.Parse(data, domain.Channel{}, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := berrymedia.Parse([]byte{}, domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
