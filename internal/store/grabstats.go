package store

import (
	"sync"
	"time"
)

type GrabStats struct {
	Source    string    `json:"source"`
	Channels int       `json:"channels"`
	Programs int       `json:"programs"`
	Errors   int       `json:"errors"`
	LastGrab time.Time `json:"last_grab"`
	Duration string    `json:"duration"`
}

type GrabStatsTracker struct {
	mu    sync.RWMutex
	stats map[string]GrabStats
}

func NewGrabStatsTracker() *GrabStatsTracker {
	return &GrabStatsTracker{
		stats: make(map[string]GrabStats),
	}
}

func (t *GrabStatsTracker) Record(source string, channels, programs, errors int, duration time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.stats[source] = GrabStats{
		Source:    source,
		Channels:  channels,
		Programs:  programs,
		Errors:    errors,
		LastGrab:  time.Now().UTC(),
		Duration:  duration.Round(time.Millisecond).String(),
	}
}

func (t *GrabStatsTracker) Get(source string) (GrabStats, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	s, ok := t.stats[source]
	return s, ok
}

func (t *GrabStatsTracker) All() []GrabStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]GrabStats, 0, len(t.stats))
	for _, s := range t.stats {
		result = append(result, s)
	}
	return result
}
