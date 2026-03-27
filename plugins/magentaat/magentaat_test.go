package magentaat_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/magentaat"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

var channel = domain.Channel{
	ID:      "magentaat-206969383991",
	XmltvID: "13thStreet.de",
	SiteID:  "206969383991",
	Lang:    "de",
}

func TestParseSchedule(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_0000.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := magentaat.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}

	p := programs[0]

	if p.Title != "So Help Me Todd" {
		t.Fatalf("expected title 'So Help Me Todd', got %q", p.Title)
	}

	expectedStart := time.Date(2023, 11, 14, 23, 20, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2023, 11, 15, 0, 5, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Category != "Kriminaldrama" {
		t.Fatalf("expected category 'Kriminaldrama', got %q", p.Category)
	}

	if p.Season != 1 {
		t.Fatalf("expected season 1, got %d", p.Season)
	}

	if p.Episode != 15 {
		t.Fatalf("expected episode 15, got %d", p.Episode)
	}

	if p.Channel != "13thStreet.de" {
		t.Fatalf("expected channel '13thStreet.de', got %q", p.Channel)
	}

	if p.Source != "tv.magenta.at" {
		t.Fatalf("expected source 'tv.magenta.at', got %q", p.Source)
	}
}

func TestParseScheduleMultipleGenres(t *testing.T) {
	data, err := os.ReadFile(testdataPath("content_0000.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := magentaat.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	last := programs[len(programs)-1]
	if last.Category != "Kriminaldrama,Drama" {
		t.Fatalf("expected category 'Kriminaldrama,Drama', got %q", last.Category)
	}
}

func TestParseDetail(t *testing.T) {
	data, err := os.ReadFile(testdataPath("program.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	detail, err := magentaat.ParseDetail(data)
	if err != nil {
		t.Fatalf("parse detail failed: %v", err)
	}

	if detail.Details == nil {
		t.Fatal("expected details to be set")
	}

	expected := "Ava ist 17 und eine geniale Hackerin. Jetzt steht die Teenagerin vor Gericht, weil sie sich illegal Zugang zum Verteidigungsministerium verschafft hat. Todd soll das IT-Genie überwachen."
	if detail.Details.Description != expected {
		t.Fatalf("expected description %q, got %q", expected, detail.Details.Description)
	}

	if detail.PosterImageURL == "" {
		t.Fatal("expected poster image URL to be set")
	}

	actors := filterRoles(detail.Roles, "Schauspieler")
	if len(actors) != 7 {
		t.Fatalf("expected 7 actors, got %d", len(actors))
	}
	if actors[0] != "Marcia Gay Harden" {
		t.Fatalf("expected first actor 'Marcia Gay Harden', got %q", actors[0])
	}

	directors := filterRoles(detail.Roles, "Regisseur")
	if len(directors) != 1 {
		t.Fatalf("expected 1 director, got %d", len(directors))
	}
	if directors[0] != "Jay Karas" {
		t.Fatalf("expected director 'Jay Karas', got %q", directors[0])
	}

	producers := filterRoles(detail.Roles, "Produzent")
	if len(producers) != 7 {
		t.Fatalf("expected 7 producers, got %d", len(producers))
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := magentaat.Parse([]byte(`{"channels":{}}`), channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := magentaat.Parse([]byte(`not json`), channel)
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseMissingChannel(t *testing.T) {
	other := domain.Channel{SiteID: "999999"}
	data, err := os.ReadFile(testdataPath("content_0000.json"))
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	programs, err := magentaat.Parse(data, other)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func filterRoles(roles []magentaat.Role, roleName string) []string {
	var names []string
	for _, r := range roles {
		if r.RoleName == roleName {
			names = append(names, r.PersonName)
		}
	}
	return names
}
