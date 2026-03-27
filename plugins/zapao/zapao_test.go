package zapao_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/zapao"
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
		SiteID:  "2275",
		XmltvID: "TPA1.ao",
		Lang:    "pt",
	}

	programs, err := zapao.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Jornal da Meia-Noite" {
		t.Fatalf("expected title 'Jornal da Meia-Noite', got %q", p.Title)
	}

	expectedStart := time.Date(2023, 5, 27, 23, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 5, 28, 0, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Category != "Noticiário" {
		t.Fatalf("expected category 'Noticiário', got %q", p.Category)
	}

	if p.Channel != "TPA1.ao" {
		t.Fatalf("expected channel 'TPA1.ao', got %q", p.Channel)
	}

	if p.Source != "zap.co.ao" {
		t.Fatalf("expected source 'zap.co.ao', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := zapao.Parse([]byte(`{"data":[]}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyArray(t *testing.T) {
	programs, err := zapao.Parse([]byte(`[]`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyBytes(t *testing.T) {
	programs, err := zapao.Parse([]byte{}, domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
