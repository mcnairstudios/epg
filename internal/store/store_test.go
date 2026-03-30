package store_test

import (
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

func TestSaveAndLoadPerSource(t *testing.T) {
	dir := t.TempDir()

	s := store.New(dir)
	now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)

	s.SetPrograms("ch1", []domain.Program{
		{Channel: "ch1", Source: "test-source", Title: "Test Show", Start: now, Stop: now.Add(time.Hour)},
	})
	s.RecordGrabStats("test-source", 1, 1, 1, 0, time.Second)

	if err := s.SaveSource("test-source"); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	s2 := store.New(dir)
	if err := s2.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	programs := s2.GetPrograms()
	if len(programs) != 1 {
		t.Fatalf("expected 1 program after load, got %d", len(programs))
	}

	stats := s2.GetGrabStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 grab stat after load, got %d", len(stats))
	}
}

func TestLazyLoadOnAccess(t *testing.T) {
	dir := t.TempDir()

	s := store.New(dir)
	now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	s.SetPrograms("ch1", []domain.Program{
		{Channel: "ch1", Source: "lazy-source", Title: "Show", Start: now, Stop: now.Add(time.Hour)},
	})
	s.RecordGrabStats("lazy-source", 1, 1, 1, 0, time.Second)
	if err := s.SaveSource("lazy-source"); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	s2 := store.New(dir)
	if err := s2.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	// After Load, stats should be available but programs not yet in memory
	if s2.ProgramCount() != 1 {
		t.Fatalf("expected ProgramCount 1 from grabStats, got %d", s2.ProgramCount())
	}

	// Accessing programs triggers lazy load
	programs := s2.GetPrograms(store.Filter{Source: "lazy-source"})
	if len(programs) != 1 {
		t.Fatalf("expected 1 program after lazy load, got %d", len(programs))
	}
}

func TestEviction(t *testing.T) {
	dir := t.TempDir()

	s := store.New(dir)
	s.SetIdleTimeout(50 * time.Millisecond)

	now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	s.SetPrograms("ch1", []domain.Program{
		{Channel: "ch1", Source: "evict-source", Title: "Show", Start: now, Stop: now.Add(time.Hour)},
	})
	s.RecordGrabStats("evict-source", 1, 1, 1, 0, time.Second)
	if err := s.SaveSource("evict-source"); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	if len(s.GetPrograms(store.Filter{Source: "evict-source"})) != 1 {
		t.Fatal("expected program before eviction")
	}

	time.Sleep(100 * time.Millisecond)
	s.EvictIdle()

	// After eviction, programs no longer in memory but lazy-load brings them back
	programs := s.GetPrograms(store.Filter{Source: "evict-source"})
	if len(programs) != 1 {
		t.Fatalf("expected program to be reloaded after eviction, got %d", len(programs))
	}

	// ProgramCount from grabStats is unaffected by eviction
	if s.ProgramCount() != 1 {
		t.Fatalf("expected ProgramCount 1 after eviction, got %d", s.ProgramCount())
	}
}

func TestLoadNonexistentDir(t *testing.T) {
	s := store.New("/tmp/nonexistent_epg_test_dir")
	if err := s.Load(); err != nil {
		t.Fatalf("load of nonexistent dir should not error, got: %v", err)
	}
}

func TestSaveEmptyDir(t *testing.T) {
	s := store.New("")
	if err := s.Save(); err != nil {
		t.Fatalf("save with empty dir should not error, got: %v", err)
	}
}

func TestMultipleSourcesSaveLoad(t *testing.T) {
	dir := t.TempDir()

	s := store.New(dir)
	now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)

	s.SetPrograms("ch1", []domain.Program{
		{Channel: "ch1", Source: "src1", Title: "Show A", Start: now, Stop: now.Add(time.Hour)},
	})
	s.SetPrograms("ch2", []domain.Program{
		{Channel: "ch2", Source: "src2", Title: "Show B", Start: now, Stop: now.Add(time.Hour)},
	})
	s.RecordGrabStats("src1", 1, 1, 1, 0, time.Second)
	s.RecordGrabStats("src2", 1, 1, 1, 0, time.Second)

	if err := s.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	s2 := store.New(dir)
	if err := s2.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if len(s2.GetPrograms()) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(s2.GetPrograms()))
	}
	if len(s2.GetGrabStats()) != 2 {
		t.Fatalf("expected 2 stats, got %d", len(s2.GetGrabStats()))
	}
}
