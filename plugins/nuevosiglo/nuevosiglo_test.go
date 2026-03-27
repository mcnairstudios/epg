package nuevosiglo_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/nuevosiglo"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseProgramDetail(t *testing.T) {
	data, err := os.ReadFile(testdataPath("program1.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	detail, err := nuevosiglo.ParseProgramDetail(data)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if detail == nil {
		t.Fatal("expected non-nil detail")
	}

	channel := domain.Channel{
		XmltvID: "HBOLatinAmerica.us",
		Lang:    "es",
	}

	_ = channel

	if detail.MainTitle != "Jurassic World: Dominion" {
		t.Fatalf("expected title 'Jurassic World: Dominion', got %q", detail.MainTitle)
	}

	if detail.Year != "2022" {
		t.Fatalf("expected year '2022', got %q", detail.Year)
	}

	if detail.ParentalRating != "PG-13" {
		t.Fatalf("expected rating 'PG-13', got %q", detail.ParentalRating)
	}

	if detail.TimeStart != "2023-02-09 22:11:00" {
		t.Fatalf("expected time_start '2023-02-09 22:11:00', got %q", detail.TimeStart)
	}
}

func TestParseProgramDetail2(t *testing.T) {
	data, err := os.ReadFile(testdataPath("program2.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	detail, err := nuevosiglo.ParseProgramDetail(data)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if detail.MainTitle != "Black Adam" {
		t.Fatalf("expected title 'Black Adam', got %q", detail.MainTitle)
	}

	montevideo := time.FixedZone("UYT", -3*60*60)
	start, _ := time.ParseInLocation("2006-01-02 15:04:05", detail.TimeStart, montevideo)
	expectedStart := time.Date(2023, 2, 11, 2, 6, 0, 0, time.UTC)
	if !start.UTC().Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, start.UTC())
	}
}

func TestParseEmptyContent(t *testing.T) {
	_, err := nuevosiglo.ParseProgramDetail([]byte(`{}`))
	if err != nil {
		t.Fatalf("parse should not fail on empty object: %v", err)
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := nuevosiglo.ParseProgramDetail([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
