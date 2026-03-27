package channels_test

import (
	"testing"

	"github.com/gavinmcnair/epg/internal/channels"
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

	if chs[1].XmltvID != "64257#6720" {
		t.Fatalf("expected fallback xmltv_id '64257#6720', got %q", chs[1].XmltvID)
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
