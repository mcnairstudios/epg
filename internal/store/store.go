package store

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
)

type Store struct {
	mu        sync.RWMutex
	channels  map[string]domain.Channel
	programs  map[string][]domain.Program
	grabStats map[string]GrabStats
	path      string
}

type snapshot struct {
	Channels  map[string]domain.Channel
	Programs  map[string][]domain.Program
	GrabStats map[string]GrabStats
}

func New(path string) *Store {
	return &Store{
		channels:  make(map[string]domain.Channel),
		programs:  make(map[string][]domain.Program),
		grabStats: make(map[string]GrabStats),
		path:      path,
	}
}

func (s *Store) SetChannels(channels []domain.Channel) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ch := range channels {
		s.channels[ch.ID] = ch
	}
}

func (s *Store) GetChannels(filters ...Filter) []domain.Channel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]domain.Channel, 0, len(s.channels))
	for _, ch := range s.channels {
		if matchesChannelFilters(ch, filters) {
			result = append(result, ch)
		}
	}
	return result
}

func (s *Store) GetChannel(id string) (domain.Channel, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch, ok := s.channels[id]
	return ch, ok
}

func (s *Store) SetPrograms(channelID string, programs []domain.Program) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.programs[channelID] = append(s.programs[channelID], programs...)
}

func (s *Store) ClearPrograms(channelID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.programs, channelID)
}

func (s *Store) GetPrograms(filters ...Filter) []domain.Program {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []domain.Program
	for _, progs := range s.programs {
		for _, p := range progs {
			if matchesProgramFilters(p, filters) {
				result = append(result, p)
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Channel != result[j].Channel {
			return result[i].Channel < result[j].Channel
		}
		return result[i].Start.Before(result[j].Start)
	})

	return result
}

func (s *Store) ChannelCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.channels)
}

func (s *Store) ProgramCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, progs := range s.programs {
		count += len(progs)
	}
	return count
}

func (s *Store) RecordGrabStats(source string, channels, programs, errors int, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.grabStats[source] = GrabStats{
		Source:   source,
		Channels: channels,
		Programs: programs,
		Errors:   errors,
		LastGrab: time.Now().UTC(),
		Duration: duration.Round(time.Millisecond).String(),
	}
}

func (s *Store) GetGrabStats() []GrabStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]GrabStats, 0, len(s.grabStats))
	for _, gs := range s.grabStats {
		result = append(result, gs)
	}
	return result
}

func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.path == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}

	f, err := os.Create(s.path)
	if err != nil {
		return err
	}
	defer f.Close()

	return gob.NewEncoder(f).Encode(snapshot{
		Channels:  s.channels,
		Programs:  s.programs,
		GrabStats: s.grabStats,
	})
}

func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.path == "" {
		return nil
	}

	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	var snap snapshot
	if err := gob.NewDecoder(f).Decode(&snap); err != nil {
		return err
	}

	s.channels = snap.Channels
	s.programs = snap.Programs
	if snap.GrabStats != nil {
		s.grabStats = snap.GrabStats
	} else {
		s.grabStats = s.deriveGrabStats()
	}
	return nil
}

func (s *Store) deriveGrabStats() map[string]GrabStats {
	stats := make(map[string]GrabStats)

	sourceChannels := make(map[string]int)
	sourcePrograms := make(map[string]int)

	for _, ch := range s.channels {
		sourceChannels[ch.Source]++
	}

	for _, progs := range s.programs {
		for _, p := range progs {
			sourcePrograms[p.Source]++
		}
	}

	sources := make(map[string]bool)
	for src := range sourceChannels {
		sources[src] = true
	}
	for src := range sourcePrograms {
		sources[src] = true
	}

	for src := range sources {
		stats[src] = GrabStats{
			Source:   src,
			Channels: sourceChannels[src],
			Programs: sourcePrograms[src],
			LastGrab: time.Now().UTC(),
			Duration: "restored",
		}
	}

	return stats
}

type Filter struct {
	Source  string
	Channel string
	Genre   string
	Quality string
	Region  string
	Date    *time.Time
	Lang    string
}

func matchesChannelFilters(ch domain.Channel, filters []Filter) bool {
	if len(filters) == 0 {
		return true
	}
	f := filters[0]
	if f.Source != "" && ch.Source != f.Source {
		return false
	}
	if f.Genre != "" && ch.Genre != f.Genre {
		return false
	}
	if f.Quality != "" && ch.Quality != f.Quality {
		return false
	}
	if f.Region != "" && ch.Region != f.Region {
		return false
	}
	if f.Lang != "" && ch.Lang != f.Lang {
		return false
	}
	if f.Channel != "" && ch.ID != f.Channel {
		return false
	}
	return true
}

func matchesProgramFilters(p domain.Program, filters []Filter) bool {
	if len(filters) == 0 {
		return true
	}
	f := filters[0]
	if f.Source != "" && p.Source != f.Source {
		return false
	}
	if f.Channel != "" && p.Channel != f.Channel {
		return false
	}
	if f.Genre != "" && p.Category != f.Genre {
		return false
	}
	if f.Date != nil {
		day := f.Date.Truncate(24 * time.Hour)
		if p.Start.Before(day) || p.Start.After(day.Add(24*time.Hour)) {
			return false
		}
	}
	return true
}
