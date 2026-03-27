package dsmart_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/dsmart"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseContent(t *testing.T) {
	data1, err := os.ReadFile(testdataPath("content1.json"))
	if err != nil {
		t.Fatalf("failed to read content1: %v", err)
	}

	data2, err := os.ReadFile(testdataPath("content2.json"))
	if err != nil {
		t.Fatalf("failed to read content2: %v", err)
	}

	var resp1, resp2 struct {
		Data struct {
			Total    int               `json:"total"`
			Channels []json.RawMessage `json:"channels"`
		} `json:"data"`
	}
	json.Unmarshal(data1, &resp1)
	json.Unmarshal(data2, &resp2)

	resp1.Data.Channels = append(resp1.Data.Channels, resp2.Data.Channels...)
	combined, _ := json.Marshal(resp1)

	channel := domain.Channel{
		ID:      "dsmart-sinema1001",
		XmltvID: "Sinema1001.tr",
		SiteID:  "5fe07f5dcfef0b1593275822",
		Lang:    "tr",
	}

	programs, err := dsmart.Parse(combined, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 11 {
		t.Fatalf("expected 11 programs, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "Taksi Şoförü" {
		t.Fatalf("unexpected title: %q", p.Title)
	}

	expectedStart := time.Date(2025, 1, 12, 21, 30, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 1, 12, 23, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	last := programs[10]
	if last.Title != "Senin Adın" {
		t.Fatalf("unexpected last title: %q", last.Title)
	}

	expectedLastStart := time.Date(2025, 1, 13, 19, 0, 0, 0, time.UTC)
	if !last.Start.Equal(expectedLastStart) {
		t.Fatalf("expected last start %v, got %v", expectedLastStart, last.Start)
	}

	expectedLastStop := time.Date(2025, 1, 13, 21, 0, 0, 0, time.UTC)
	if !last.Stop.Equal(expectedLastStop) {
		t.Fatalf("expected last stop %v, got %v", expectedLastStop, last.Stop)
	}
}

func TestParseEmptyContent(t *testing.T) {
	data, err := os.ReadFile(testdataPath("no_content.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	channel := domain.Channel{SiteID: "test"}
	programs, err := dsmart.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := dsmart.Parse([]byte("not json"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse should not error, got: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
