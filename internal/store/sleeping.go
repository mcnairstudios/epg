package store

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"sync"
)

type SleepingStore struct {
	mu       sync.RWMutex
	sleeping map[string]string
	path     string
}

func NewSleepingStore(path string) *SleepingStore {
	return &SleepingStore{
		sleeping: make(map[string]string),
		path:     path,
	}
}

func (s *SleepingStore) Sleep(name, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sleeping[name] = reason
}

func (s *SleepingStore) Wake(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sleeping, name)
}

func (s *SleepingStore) IsSleeping(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.sleeping[name]
	return ok
}

func (s *SleepingStore) List() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]string, len(s.sleeping))
	for k, v := range s.sleeping {
		result[k] = v
	}
	return result
}

func (s *SleepingStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sleeping)
}

func (s *SleepingStore) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.path == "" {
		return nil
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".sleeping-*.gob.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if err := gob.NewEncoder(tmp).Encode(s.sleeping); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	tmp.Sync()
	tmp.Close()

	return os.Rename(tmpPath, s.path)
}

func (s *SleepingStore) Load() error {
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

	return gob.NewDecoder(f).Decode(&s.sleeping)
}
