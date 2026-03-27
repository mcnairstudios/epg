package source_test

import (
	"path/filepath"
	"testing"

	"github.com/gavinmcnair/epg/internal/source"
)

func TestAddAndList(t *testing.T) {
	m := source.NewExternalSourceManager("")

	if err := m.Add("uk-epg", "https://example.com/epg_uk.xml"); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	sources := m.List()
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}

	if sources[0].Name != "uk-epg" {
		t.Fatalf("expected name 'uk-epg', got %q", sources[0].Name)
	}
}

func TestAddDuplicate(t *testing.T) {
	m := source.NewExternalSourceManager("")

	m.Add("test", "https://example.com/test.xml")
	if err := m.Add("test", "https://example.com/test2.xml"); err == nil {
		t.Fatal("expected error on duplicate add")
	}
}

func TestRemove(t *testing.T) {
	m := source.NewExternalSourceManager("")

	m.Add("test", "https://example.com/test.xml")
	if err := m.Remove("test"); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	sources := m.List()
	if len(sources) != 0 {
		t.Fatalf("expected 0 sources after remove, got %d", len(sources))
	}
}

func TestRemoveNonexistent(t *testing.T) {
	m := source.NewExternalSourceManager("")

	if err := m.Remove("nonexistent"); err == nil {
		t.Fatal("expected error on nonexistent remove")
	}
}

func TestGet(t *testing.T) {
	m := source.NewExternalSourceManager("")

	m.Add("test", "https://example.com/test.xml")

	s, ok := m.Get("test")
	if !ok {
		t.Fatal("expected source to be found")
	}
	if s.URL != "https://example.com/test.xml" {
		t.Fatalf("expected URL, got %q", s.URL)
	}

	_, ok = m.Get("nonexistent")
	if ok {
		t.Fatal("expected source not to be found")
	}
}

func TestGzipDetection(t *testing.T) {
	m := source.NewExternalSourceManager("")

	m.Add("gz-source", "https://example.com/epg.xml.gz")

	s, _ := m.Get("gz-source")
	if s.Type != "xmltv-gz" {
		t.Fatalf("expected type 'xmltv-gz', got %q", s.Type)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sources.gob")

	m := source.NewExternalSourceManager(path)
	m.Add("test", "https://example.com/test.xml")
	m.Add("gz-test", "https://example.com/test.xml.gz")

	if err := m.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	m2 := source.NewExternalSourceManager(path)
	if err := m2.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	sources := m2.List()
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources after load, got %d", len(sources))
	}
}

func TestLoadNonexistent(t *testing.T) {
	m := source.NewExternalSourceManager("/tmp/nonexistent_sources.gob")
	if err := m.Load(); err != nil {
		t.Fatalf("load of nonexistent should not error: %v", err)
	}
}
