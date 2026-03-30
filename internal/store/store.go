package store

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
)

const DefaultIdleTimeout = 30 * time.Minute

type GrabStats struct {
	Source        string    `json:"source"`
	TotalChannels int       `json:"total_channels"`
	Channels      int       `json:"channels"`
	Programs      int       `json:"programs"`
	Errors        int       `json:"errors"`
	LastGrab      time.Time `json:"last_grab"`
	Duration      string    `json:"duration"`
}

type Store struct {
	mu             sync.RWMutex
	channels       map[string]domain.Channel
	programs       map[string][]domain.Program
	grabStats      map[string]GrabStats
	sourceChannels map[string]map[string]bool
	loaded         map[string]bool
	lastAccess     map[string]time.Time
	dir            string
	idleTimeout    time.Duration
	stopCh         chan struct{}
}

type sourceSnapshot struct {
	Programs  map[string][]domain.Program
	GrabStats GrabStats
}

func New(dir string) *Store {
	s := &Store{
		channels:       make(map[string]domain.Channel),
		programs:       make(map[string][]domain.Program),
		grabStats:      make(map[string]GrabStats),
		sourceChannels: make(map[string]map[string]bool),
		loaded:         make(map[string]bool),
		lastAccess:     make(map[string]time.Time),
		dir:            dir,
		idleTimeout:    DefaultIdleTimeout,
		stopCh:         make(chan struct{}),
	}
	if dir != "" {
		go s.evictLoop()
	}
	return s
}

func (s *Store) Close() {
	close(s.stopCh)
}

func (s *Store) SetIdleTimeout(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.idleTimeout = d
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

	if len(programs) > 0 && programs[0].Source != "" {
		source := programs[0].Source
		if s.sourceChannels[source] == nil {
			s.sourceChannels[source] = make(map[string]bool)
		}
		s.sourceChannels[source][channelID] = true
		s.loaded[source] = true
		s.lastAccess[source] = time.Now()
	}
}

func (s *Store) ClearPrograms(channelID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.programs, channelID)
}

func (s *Store) GetPrograms(filters ...Filter) []domain.Program {
	source := ""
	if len(filters) > 0 {
		source = filters[0].Source
	}

	s.mu.Lock()
	if source != "" {
		s.ensureLoaded(source)
		s.lastAccess[source] = time.Now()
	} else {
		for src := range s.grabStats {
			s.ensureLoaded(src)
			s.lastAccess[src] = time.Now()
		}
	}
	s.mu.Unlock()

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

// ProgramCount returns the total programs across all sources from grab stats.
// This is always accurate without needing programs loaded in memory.
func (s *Store) ProgramCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	total := 0
	for _, gs := range s.grabStats {
		total += gs.Programs
	}
	return total
}

func (s *Store) RecordGrabStats(source string, totalChannels, channels, programs, errors int, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.grabStats[source] = GrabStats{
		Source:        source,
		TotalChannels: totalChannels,
		Channels:      channels,
		Programs:      programs,
		Errors:        errors,
		LastGrab:      time.Now().UTC(),
		Duration:      duration.Round(time.Millisecond).String(),
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

func (s *Store) SaveSource(source string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.dir == "" {
		return nil
	}

	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return err
	}

	progs := make(map[string][]domain.Program)
	for chID, p := range s.programs {
		if len(p) > 0 && p[0].Source == source {
			progs[chID] = p
		}
	}

	snap := sourceSnapshot{
		Programs:  progs,
		GrabStats: s.grabStats[source],
	}

	safeName := strings.ReplaceAll(source, "/", "_")
	target := filepath.Join(s.dir, safeName+".gob")

	tmp, err := os.CreateTemp(s.dir, "."+safeName+"-*.gob.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if err := gob.NewEncoder(tmp).Encode(snap); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	tmp.Close()

	return os.Rename(tmpPath, target)
}

func (s *Store) Save() error {
	s.mu.RLock()
	sources := make(map[string]bool)
	for _, gs := range s.grabStats {
		sources[gs.Source] = true
	}
	s.mu.RUnlock()

	for source := range sources {
		if err := s.SaveSource(source); err != nil {
			return err
		}
	}
	return nil
}

// Load reads GrabStats from all source gob files but does not load programs.
// Programs are loaded lazily on first access.
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dir == "" {
		return nil
	}

	files, err := filepath.Glob(filepath.Join(s.dir, "*.gob"))
	if err != nil {
		return err
	}

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}

		var snap sourceSnapshot
		if err := gob.NewDecoder(f).Decode(&snap); err != nil {
			f.Close()
			continue
		}
		f.Close()

		if snap.GrabStats.Source != "" {
			s.grabStats[snap.GrabStats.Source] = snap.GrabStats
		}
	}

	return nil
}

// ensureLoaded loads programs for a source from disk if not already in memory.
// Must be called with the write lock held.
func (s *Store) ensureLoaded(source string) {
	if s.loaded[source] || s.dir == "" {
		return
	}

	safeName := strings.ReplaceAll(source, "/", "_")
	path := filepath.Join(s.dir, safeName+".gob")

	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	var snap sourceSnapshot
	if err := gob.NewDecoder(f).Decode(&snap); err != nil {
		return
	}

	if s.sourceChannels[source] == nil {
		s.sourceChannels[source] = make(map[string]bool)
	}

	for chID, progs := range snap.Programs {
		s.programs[chID] = progs
		s.sourceChannels[source][chID] = true
	}

	s.loaded[source] = true
}

func (s *Store) evictLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.EvictIdle()
		}
	}
}

func (s *Store) EvictIdle() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for source, lastAccess := range s.lastAccess {
		if now.Sub(lastAccess) > s.idleTimeout {
			for chID := range s.sourceChannels[source] {
				delete(s.programs, chID)
			}
			delete(s.loaded, source)
			delete(s.lastAccess, source)
		}
	}
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
