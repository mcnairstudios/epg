package meopt_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/meopt"
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
		ID:      "meopt-RTPM",
		XmltvID: "RTPMadeira.pt",
		SiteID:  "RTPM",
		Lang:    "pt",
	}

	programs, err := meopt.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "Barman - Ep. 4" {
		t.Fatalf("expected title 'Barman - Ep. 4', got %q", p.Title)
	}

	expectedStart := time.Date(2026, 1, 22, 23, 40, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2026, 1, 23, 0, 4, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	expectedDesc := "'Barman' é uma série de comédia dramática sobre um jovem comediante que começa a trabalhar como Barman porque precisa de arranjar dinheiro depressa, pelo caminho é obrigado a lidar com a vida noturna e conciliar duas realidades diferentes."
	if p.Description != expectedDesc {
		t.Fatalf("unexpected description: %q", p.Description)
	}

	expectedImage := "https://cdn-er-images.online.meo.pt/eemstb/ImageHandler.ashx?chCallLetter=RTPM&progTitle=Barman+-+Ep.+4&profile=16_9&profileFallback=false&noFallback=true&appSource=PC_CHROME_PWA&width=1920&csf"
	if p.Image != expectedImage {
		t.Fatalf("expected image %q, got %q", expectedImage, p.Image)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := meopt.Parse([]byte(""), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := meopt.Parse([]byte("not json"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
