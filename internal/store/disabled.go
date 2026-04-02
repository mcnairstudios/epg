package store

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"sync"
)

type DisabledStore struct {
	mu       sync.RWMutex
	disabled map[string]bool
	path     string
}

func NewDisabledStore(path string) *DisabledStore {
	return &DisabledStore{
		disabled: make(map[string]bool),
		path:     path,
	}
}

func (s *DisabledStore) Disable(name string) error {
	s.mu.Lock()
	s.disabled[name] = true
	s.mu.Unlock()
	return s.Save()
}

func (s *DisabledStore) Enable(name string) error {
	s.mu.Lock()
	delete(s.disabled, name)
	s.mu.Unlock()
	return s.Save()
}

func (s *DisabledStore) IsDisabled(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.disabled[name]
}

func (s *DisabledStore) List() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]bool, len(s.disabled))
	for k := range s.disabled {
		result[k] = true
	}
	return result
}

func (s *DisabledStore) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.path == "" {
		return nil
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".disabled-*.gob.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if err := gob.NewEncoder(tmp).Encode(s.disabled); err != nil {
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

	return os.Rename(tmpPath, s.path)
}

func (s *DisabledStore) Load() error {
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

	return gob.NewDecoder(f).Decode(&s.disabled)
}
