package rtsch_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/rtsch"
)

func TestParseContent(t *testing.T) {
	content := map[string]interface{}{
		"programGuide": []interface{}{
			map[string]interface{}{
				"channel": map[string]interface{}{
					"id":    "5d332a26e06d08eec8ad385d566187df72955623",
					"title": "RTS Info",
				},
				"programList": []interface{}{
					map[string]interface{}{
						"title":     "L'essentiel de l'actualite",
						"startTime": "2026-03-21T07:00:00+01:00",
						"endTime":   "2026-03-21T19:00:00+01:00",
						"imageUrl":  "https://kingfisher.rts.ch/res/img/cdns3/sherlock/urn:orphea-image:1043433",
						"genre":     "Actualite",
					},
					map[string]interface{}{
						"title":       "Forum",
						"startTime":   "2026-03-21T19:00:00+01:00",
						"endTime":     "2026-03-21T20:00:00+01:00",
						"imageUrl":    "https://kingfisher.rts.ch/res/img/cdns3/sherlock/urn:orphea-image:1831387",
						"genre":       "Actualite",
						"description": "Le magazine du soir.",
					},
				},
			},
		},
	}

	data, _ := json.Marshal(content)
	channel := domain.Channel{
		XmltvID: "RTSInfo.ch",
		SiteID:  "5d332a26e06d08eec8ad385d566187df72955623",
		Lang:    "fr",
	}

	programs, err := rtsch.Parse(data, channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}

	if programs[0].Title != "L'essentiel de l'actualite" {
		t.Fatalf("unexpected title: %q", programs[0].Title)
	}

	expectedStart := time.Date(2026, 3, 21, 6, 0, 0, 0, time.UTC)
	if !programs[0].Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, programs[0].Start)
	}

	expectedStop := time.Date(2026, 3, 21, 18, 0, 0, 0, time.UTC)
	if !programs[0].Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, programs[0].Stop)
	}

	if programs[0].Category != "Actualite" {
		t.Fatalf("expected category 'Actualite', got %q", programs[0].Category)
	}

	if programs[1].Description != "Le magazine du soir." {
		t.Fatalf("expected description 'Le magazine du soir.', got %q", programs[1].Description)
	}
}

func TestParseEmptyProgramList(t *testing.T) {
	content := `{"programGuide":[{"channel":{"id":"x","title":"X"},"programList":[]}]}`
	programs, err := rtsch.Parse([]byte(content), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseEmptyGuide(t *testing.T) {
	programs, err := rtsch.Parse([]byte(""), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseMalformedJSON(t *testing.T) {
	programs, err := rtsch.Parse([]byte("not-json"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
