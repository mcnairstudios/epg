package rtppt_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/rtppt"
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
		ID:      "rtppt-lis-4",
		XmltvID: "RTPMadeira.pt",
		SiteID:  "lis#4",
		Lang:    "pt",
	}

	programs, err := rtppt.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	found := false
	for _, p := range programs {
		if p.Title == "Telejornal Madeira" && p.Start.Format("2006-01-02T15:04:05.000Z") == "2022-12-02T23:30:00.000Z" {
			found = true
			if p.Stop.Format("2006-01-02T15:04:05.000Z") != "2022-12-03T00:00:00.000Z" {
				t.Fatalf("unexpected stop: %v", p.Stop)
			}
			if p.Description != "Informação de proximidade. De confiança!" {
				t.Fatalf("unexpected description: %q", p.Description)
			}
			if p.Image != "https://cdn-images.rtp.pt/EPG/imagens/15790_43438_8820.png?w=384&h=216" {
				t.Fatalf("unexpected image: %q", p.Image)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected to find 'Telejornal Madeira' at 23:30")
	}
}

func TestParseEmptyContent(t *testing.T) {
	channel := domain.Channel{SiteID: "lis#4"}
	programs, err := rtppt.Parse([]byte(""), channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	channel := domain.Channel{SiteID: "lis#4"}
	programs, err := rtppt.Parse([]byte(`not json`), channel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
