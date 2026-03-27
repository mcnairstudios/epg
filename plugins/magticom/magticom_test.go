package magticom_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/magticom"
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
		ID:      "magticom-260",
		XmltvID: "BollywoodHDRussia.ru",
		SiteID:  "260",
		Lang:    "ka",
	}

	programs, err := magticom.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Х/ф \"Неравный брак\"." {
		t.Fatalf("unexpected title: %q", p.Title)
	}

	tbilisi, _ := time.LoadLocation("Asia/Tbilisi")
	expectedStart := time.Date(2021, 11, 22, 7, 0, 0, 0, tbilisi)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2021, 11, 22, 9, 0, 0, 0, tbilisi)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "BollywoodHDRussia.ru" {
		t.Fatalf("expected channel BollywoodHDRussia.ru, got %q", p.Channel)
	}

	if p.Source != "magticom.ge" {
		t.Fatalf("expected source magticom.ge, got %q", p.Source)
	}

	if p.Description == "" {
		t.Fatal("expected non-empty description")
	}
}

func TestParseEmptyArray(t *testing.T) {
	programs, err := magticom.Parse([]byte(`[]`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := magticom.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
