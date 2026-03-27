package hoytv_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/hoytv"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content.xml"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{
		ID:      "hoytv-76",
		XmltvID: "HOYIBC.hk",
		SiteID:  "76",
		Lang:    "zh",
	}

	date := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)

	programs, err := hoytv.Parse(data, channel, date)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) < 2 {
		t.Fatalf("expected at least 2 programs, got %d", len(programs))
	}

	p := programs[0]

	expectedStart := time.Date(2024, 9, 13, 3, 30, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2024, 9, 13, 4, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.SubTitle != "第46集" {
		t.Fatalf("unexpected subtitle: %q", p.SubTitle)
	}
}

func TestParseInvalidXML(t *testing.T) {
	_, err := hoytv.Parse([]byte("not xml"), domain.Channel{}, time.Now())
	if err == nil {
		t.Fatal("expected error on invalid XML")
	}
}
