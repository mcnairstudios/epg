package store_test

import (
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/store"
)

func TestGrabStatsRecord(t *testing.T) {
	tracker := store.NewGrabStatsTracker()

	tracker.Record("pluto.tv", 1, 75, 0, 500*time.Millisecond)

	s, ok := tracker.Get("pluto.tv")
	if !ok {
		t.Fatal("expected stats for pluto.tv")
	}
	if s.Programs != 75 {
		t.Fatalf("expected 75 programs, got %d", s.Programs)
	}
	if s.Channels != 1 {
		t.Fatalf("expected 1 channel, got %d", s.Channels)
	}
	if s.Errors != 0 {
		t.Fatalf("expected 0 errors, got %d", s.Errors)
	}
}

func TestGrabStatsAll(t *testing.T) {
	tracker := store.NewGrabStatsTracker()

	tracker.Record("src1", 1, 10, 0, time.Second)
	tracker.Record("src2", 2, 20, 1, 2*time.Second)

	all := tracker.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 stats, got %d", len(all))
	}
}

func TestGrabStatsGetMissing(t *testing.T) {
	tracker := store.NewGrabStatsTracker()

	_, ok := tracker.Get("nonexistent")
	if ok {
		t.Fatal("expected no stats for nonexistent")
	}
}
