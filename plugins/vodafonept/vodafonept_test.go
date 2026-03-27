package vodafonept_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/vodafonept"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	files := []string{"content_0006.json", "content_0612.json", "content_1218.json", "content_1824.json"}
	var allData [][]byte
	for _, f := range files {
		data, err := os.ReadFile(testdataPath(f))
		if err != nil {
			t.Fatalf("failed to read test data %s: %v", f, err)
		}
		allData = append(allData, data)
	}

	channel := domain.Channel{
		SiteID:  "3028",
		XmltvID: "RTP1.pt",
		Lang:    "pt",
	}

	programs, err := vodafonept.ParseMulti(allData, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 15 {
		t.Fatalf("expected 15 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Balacobaco" {
		t.Fatalf("expected title 'Balacobaco', got %q", p.Title)
	}

	expectedStart := time.Date(2025, 9, 24, 0, 19, 18, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 9, 24, 1, 15, 1, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	last := programs[14]
	if last.Title != "Janela Indiscreta" {
		t.Fatalf("expected last title 'Janela Indiscreta', got %q", last.Title)
	}

	expectedLastStart := time.Date(2025, 9, 24, 23, 28, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := vodafonept.Parse([]byte("{}"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseSinglePeriod(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_0006.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		SiteID:  "3028",
		XmltvID: "RTP1.pt",
		Lang:    "pt",
	}

	programs, err := vodafonept.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}
}
