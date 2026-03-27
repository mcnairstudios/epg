package clarotvmais_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/clarotvmais"
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
		ID:      "clarotvmais-1",
		XmltvID: "DiscoveryKids.br",
		SiteID:  "1",
		Lang:    "pt",
	}

	programs, err := clarotvmais.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	if programs[0].Source != "clarotvmais.com.br" {
		t.Fatalf("expected source 'clarotvmais.com.br', got %q", programs[0].Source)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := clarotvmais.Parse([]byte(`not json`), domain.Channel{})
	if err != nil {
		t.Fatalf("expected no error on invalid JSON, got %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmpty(t *testing.T) {
	programs, err := clarotvmais.Parse([]byte(`{"response":{"liveChannels":[]}}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
