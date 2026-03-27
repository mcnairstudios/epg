package myafn_test

import (
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/plugins/myafn"
)

func TestParseContent(t *testing.T) {
	content := `[{"a":566,"b":2,"c":"2022,9,3,3,0,0,0","d":"2022,9,3,4,0,0,0","e":"2022,9,3,3,0,0,0","f":"2022,9,3,4,0,0,0","g":60,"h":"This Week with George Stephanopoulos (ABC)","i":"Episode Title","j":"TV-14","k":false,"l":"Former Clinton White House staffer.","m":"News,Politics,Public affairs,Talk","n":694284445,"o":60,"p":20,"q":true,"r":694285705,"s":null}]`

	channel := domain.Channel{
		XmltvID: "AFNPrimeAtlantic.us",
		SiteID:  "2",
	}

	programs, err := myafn.Parse([]byte(content), channel)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}

	p := programs[0]
	if p.Title != "This Week with George Stephanopoulos (ABC)" {
		t.Fatalf("expected title 'This Week with George Stephanopoulos (ABC)', got %q", p.Title)
	}

	if p.SubTitle != "Episode Title" {
		t.Fatalf("expected subtitle 'Episode Title', got %q", p.SubTitle)
	}

	expectedStart := time.Date(2022, 10, 3, 3, 0, 0, 0, time.UTC)
	if !p.Start.Equal(expectedStart) {
		t.Fatalf("expected start %v, got %v", expectedStart, p.Start)
	}

	expectedStop := time.Date(2022, 10, 3, 4, 0, 0, 0, time.UTC)
	if !p.Stop.Equal(expectedStop) {
		t.Fatalf("expected stop %v, got %v", expectedStop, p.Stop)
	}

	if p.Category != "News,Politics,Public affairs,Talk" {
		t.Fatalf("expected category 'News,Politics,Public affairs,Talk', got %q", p.Category)
	}
}

func TestParseEmptyContent(t *testing.T) {
	programs, err := myafn.Parse([]byte(`{"Message":"An error has occurred."}`), domain.Channel{SiteID: "2"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}

func TestParseInvalidJSON(t *testing.T) {
	programs, err := myafn.Parse([]byte("not json"), domain.Channel{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(programs) != 0 {
		t.Fatalf("expected 0 programs, got %d", len(programs))
	}
}
