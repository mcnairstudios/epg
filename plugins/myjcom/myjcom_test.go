package myjcom_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/myjcom"
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
		ID:      "tvguide.myjcom.jp-120_200_4",
		XmltvID: "StarChannel1.jp",
		SiteID:  "120_200_4",
		Lang:    "ja",
	}

	date := time.Date(2022, 1, 14, 0, 0, 0, 0, time.UTC)
	programs, err := myjcom.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "[5.1]フードロア：タマリンド" {
		t.Fatalf("expected title '[5.1]フードロア：タマリンド', got %q", p.Title)
	}

	expectedStart := time.Date(2022, 1, 13, 20, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 1, 13, 21, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Image != "https://tvguide.myjcom.jp/monomedia/si/2022/20220114/7305523/image/7743d17b655b8d2274ca58b74f2f095c.jpg" {
		t.Fatalf("expected image URL, got %q", p.Image)
	}

	if p.Category != "ドラマ" {
		t.Fatalf("expected category 'ドラマ', got %q", p.Category)
	}

	if p.Channel != "StarChannel1.jp" {
		t.Fatalf("expected channel 'StarChannel1.jp', got %q", p.Channel)
	}

	if p.Source != "tvguide.myjcom.jp" {
		t.Fatalf("expected source 'tvguide.myjcom.jp', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID: "120_200_4",
	}
	date := time.Date(2022, 1, 14, 0, 0, 0, 0, time.UTC)
	programs, err := myjcom.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := myjcom.Parse([]byte(`not json`), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
