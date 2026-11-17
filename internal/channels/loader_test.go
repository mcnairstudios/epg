package channels_test

import (
	"testing"

	"github.com/gavinmcnair/epg/internal/channels"
	"github.com/gavinmcnair/epg/internal/domain"
)

func TestLoad(t *testing.T) {
	data := []byte(`[
		{"site_id": "64257#4416", "name": "BBC Red Button", "lang": "en", "xmltv_id": "BBCRedButton.uk"},
		{"site_id": "64257#6720", "name": "BBC Radio 1", "lang": "en"}
	]`)

	chs, err := channels.Load(data, "freeview.co.uk")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if len(chs) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(chs))
	}

	if chs[0].Name != "BBC Red Button" {
		t.Fatalf("expected 'BBC Red Button', got %q", chs[0].Name)
	}
	if chs[0].XmltvID != "BBCRedButton.uk" {
		t.Fatalf("expected xmltv_id 'BBCRedButton.uk', got %q", chs[0].XmltvID)
	}
	if chs[0].Source != "freeview.co.uk" {
		t.Fatalf("expected source 'freeview.co.uk', got %q", chs[0].Source)
	}
	if chs[0].SiteID != "64257#4416" {
		t.Fatalf("expected site_id '64257#4416', got %q", chs[0].SiteID)
	}

	if chs[1].XmltvID != "" {
		t.Fatalf("expected empty xmltv_id for channel with no xmltv_id set, got %q", chs[1].XmltvID)
	}
}

func TestLoadEmpty(t *testing.T) {
	chs, err := channels.Load([]byte(`[]`), "test")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(chs) != 0 {
		t.Fatalf("expected 0, got %d", len(chs))
	}
}

func TestLoadInvalid(t *testing.T) {
	_, err := channels.Load([]byte("not json"), "test")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGroupByIsStableAndAdjacent(t *testing.T) {
	in := []domain.Channel{
		{ID: "1", SiteID: "b#1"}, {ID: "2", SiteID: "a#1"}, {ID: "3", SiteID: "b#2"}, {ID: "4", SiteID: "a#2"},
	}
	out := channels.GroupBy(in, func(c domain.Channel) string { return c.SiteID[:1] })
	got := ""
	for _, c := range out {
		got += c.ID
	}
	if got != "2413" {
		t.Fatalf("got %s want 2413", got)
	}
}
