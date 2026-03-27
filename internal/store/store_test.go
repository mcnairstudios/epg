package store_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/store"
)

func TestSetAndGetChannels(t *testing.T) {
	s := store.New("")

	channels := []domain.Channel{
		{ID: "ch1", Name: "Channel 1", Source: "test", Lang: "en"},
		{ID: "ch2", Name: "Channel 2", Source: "test", Lang: "es"},
	}
	s.SetChannels(channels)

	got := s.GetChannels()
	if len(got) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(got))
	}
}

func TestGetChannelsWithFilter(t *testing.T) {
	s := store.New("")

	s.SetChannels([]domain.Channel{
		{ID: "ch1", Name: "Channel 1", Source: "src1", Lang: "en"},
		{ID: "ch2", Name: "Channel 2", Source: "src2", Lang: "es"},
		{ID: "ch3", Name: "Channel 3", Source: "src1", Lang: "en", Genre: "news"},
	})

	got := s.GetChannels(store.Filter{Source: "src1"})
	if len(got) != 2 {
		t.Fatalf("expected 2 channels for src1, got %d", len(got))
	}

	got = s.GetChannels(store.Filter{Genre: "news"})
	if len(got) != 1 {
		t.Fatalf("expected 1 news channel, got %d", len(got))
	}
}

func TestGetChannel(t *testing.T) {
	s := store.New("")

	s.SetChannels([]domain.Channel{
		{ID: "ch1", Name: "Channel 1"},
	})

	ch, ok := s.GetChannel("ch1")
	if !ok {
		t.Fatal("expected channel to be found")
	}
	if ch.Name != "Channel 1" {
		t.Fatalf("expected Channel 1, got %s", ch.Name)
	}

	_, ok = s.GetChannel("nonexistent")
	if ok {
		t.Fatal("expected channel not to be found")
	}
}

func TestSetAndGetPrograms(t *testing.T) {
	s := store.New("")

	now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	programs := []domain.Program{
		{Channel: "ch1", Title: "News", Start: now, Stop: now.Add(time.Hour)},
		{Channel: "ch1", Title: "Movie", Start: now.Add(time.Hour), Stop: now.Add(2 * time.Hour)},
	}
	s.SetPrograms("ch1", programs)

	got := s.GetPrograms()
	if len(got) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(got))
	}
}

func TestGetProgramsWithFilter(t *testing.T) {
	s := store.New("")

	now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	s.SetPrograms("ch1", []domain.Program{
		{Channel: "ch1", Source: "src1", Title: "News", Category: "news", Start: now, Stop: now.Add(time.Hour)},
	})
	s.SetPrograms("ch2", []domain.Program{
		{Channel: "ch2", Source: "src2", Title: "Movie", Category: "film", Start: now, Stop: now.Add(2 * time.Hour)},
	})

	got := s.GetPrograms(store.Filter{Source: "src1"})
	if len(got) != 1 {
		t.Fatalf("expected 1 program for src1, got %d", len(got))
	}

	got = s.GetPrograms(store.Filter{Channel: "ch2"})
	if len(got) != 1 {
		t.Fatalf("expected 1 program for ch2, got %d", len(got))
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "epg.gob")

	s := store.New(path)
	s.SetChannels([]domain.Channel{
		{ID: "ch1", Name: "Channel 1", Source: "test"},
	})
	now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	s.SetPrograms("ch1", []domain.Program{
		{Channel: "ch1", Title: "Test Show", Start: now, Stop: now.Add(time.Hour)},
	})

	if err := s.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	s2 := store.New(path)
	if err := s2.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	channels := s2.GetChannels()
	if len(channels) != 1 {
		t.Fatalf("expected 1 channel after load, got %d", len(channels))
	}

	programs := s2.GetPrograms()
	if len(programs) != 1 {
		t.Fatalf("expected 1 program after load, got %d", len(programs))
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	s := store.New("/tmp/nonexistent_epg_test.gob")
	if err := s.Load(); err != nil {
		t.Fatalf("load of nonexistent file should not error, got: %v", err)
	}
}

func TestSaveEmptyPath(t *testing.T) {
	s := store.New("")
	if err := s.Save(); err != nil {
		t.Fatalf("save with empty path should not error, got: %v", err)
	}
}

func TestLoadEmptyPath(t *testing.T) {
	s := store.New("")
	if err := s.Load(); err != nil {
		t.Fatalf("load with empty path should not error, got: %v", err)
	}
}

func TestCleanup(t *testing.T) {
	os.Remove("/tmp/nonexistent_epg_test.gob")
}
