package cytacy_test

import (
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/cytacy"
)

func TestParseContent(t *testing.T) {
	data := []byte(`{
		"channelEpgs": [
			{
				"epgPlayables": [
					{ "name": "Πρώτη Ενημέρωση", "startTime": 1735879500000, "endTime": 1735889400000 }
				]
			}
		]
	}`)

	channel := domain.Channel{
		ID:      "cyta.com.cy-561066",
		XmltvID: "RIK1.cy",
		SiteID:  "561066",
		Lang:    "el",
	}

	programs, err := cytacy.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]

	if p.Title != "Πρώτη Ενημέρωση" {
		t.Fatalf("expected title 'Πρώτη Ενημέρωση', got %q", p.Title)
	}

	expectedStart := time.Date(2025, 1, 3, 4, 45, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2025, 1, 3, 7, 30, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Channel != "RIK1.cy" {
		t.Fatalf("expected channel 'RIK1.cy', got %q", p.Channel)
	}

	if p.Source != "cyta.com.cy" {
		t.Fatalf("expected source 'cyta.com.cy', got %q", p.Source)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := cytacy.Parse([]byte(`{"channelEpgs":[]}`), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	_, err := cytacy.Parse([]byte(`not json`), domain.Channel{})
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}
