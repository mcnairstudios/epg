package store

import (
	"encoding/gob"
	"os"
	"sync"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
)

type Store struct {
	mu       sync.RWMutex
	channels map[string]domain.Channel
	programs map[string][]domain.Program
	path     string
}

type snapshot struct {
	Channels map[string]domain.Channel
	Programs map[string][]domain.Program
}

func New(path string) *Store {
	return &Store{
		channels: make(map[string]domain.Channel),
		programs: make(map[string][]domain.Program),
		path:     path,
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

	s.programs[channelID] = programs
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
	return result
}

func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.path == "" {
		return nil
	}

	f, err := os.Create(s.path)
	if err != nil {
		return err
	}
	defer f.Close()

	return gob.NewEncoder(f).Encode(snapshot{
		Channels: s.channels,
		Programs: s.programs,
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
	return nil
}

type Filter struct {
	Source  string
	Channel string
	Genre   string
	Quality string
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
