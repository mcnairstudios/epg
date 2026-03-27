package skyperfect_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/skyperfect"
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

	channel := domain.Channel{
		ID:      "skyperfectv.co.jp-basic_CS333",
		XmltvID: "AnimeTheaterX.jp",
		SiteID:  "basic_CS333",
		Lang:    "ja",
	}

	date := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)
	programs, err := skyperfect.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	found := false
	for _, p := range programs {
		if p.Title == "ヴァルキリードライヴマーメイド #06" {
			found = true

			expectedStart := time.Date(2024, 7, 31, 19, 0, 0, 0, time.UTC)
			if !p.Start.Equal(expectedStart) {
				t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
			}

			expectedStop := time.Date(2024, 7, 31, 19, 30, 0, 0, time.UTC)
			if !p.Stop.Equal(expectedStop) {
				t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
			}

			if p.Image != "https://pm-img-ap.skyperfectv.co.jp/uploads/thumbnail/image/11301805/S_BC929697780313_be7975d4e26a4cad9b89fc6c94807e38_20240613144158569.jpg" {
				t.Fatalf("expected image URL, got %q", p.Image)
			}

			if p.Channel != "AnimeTheaterX.jp" {
				t.Fatalf("expected channel 'AnimeTheaterX.jp', got %q", p.Channel)
			}

			if p.Source != "skyperfectv.co.jp" {
				t.Fatalf("expected source 'skyperfectv.co.jp', got %q", p.Source)
			}
			break
		}
	}

	if !found {
		t.Fatal("expected to find program 'ヴァルキリードライヴマーメイド #06'")
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("empty.html"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := skyperfect.Parse(data, domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyData(t *testing.T) {
	programs, err := skyperfect.Parse([]byte(``), domain.Channel{}, time.Now())
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
