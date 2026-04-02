package source

import (
	"compress/gzip"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/xmltv"
)

type ExternalSource struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

type ExternalSourceManager struct {
	mu      sync.RWMutex
	sources map[string]ExternalSource
	client  *http.Client
	path    string
}

type externalSnapshot struct {
	Sources map[string]ExternalSource
}

func NewExternalSourceManager(path string) *ExternalSourceManager {
	return &ExternalSourceManager{
		sources: make(map[string]ExternalSource),
		client: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				DisableCompression: true,
			},
		},
		path: path,
	}
}

func (m *ExternalSourceManager) Add(name, url string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sources[name]; exists {
		return fmt.Errorf("source already exists: %s", name)
	}

	sourceType := "xmltv"
	if strings.HasSuffix(url, ".gz") || strings.HasSuffix(strings.TrimPrefix(url, "file://"), ".gz") {
		sourceType = "xmltv-gz"
	}

	m.sources[name] = ExternalSource{
		Name: name,
		URL:  url,
		Type: sourceType,
	}
	return nil
}

func (m *ExternalSourceManager) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sources[name]; !exists {
		return fmt.Errorf("source not found: %s", name)
	}
	delete(m.sources, name)
	return nil
}

func (m *ExternalSourceManager) List() []ExternalSource {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]ExternalSource, 0, len(m.sources))
	for _, s := range m.sources {
		result = append(result, s)
	}
	return result
}

func (m *ExternalSourceManager) Get(name string) (ExternalSource, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.sources[name]
	return s, ok
}

func (m *ExternalSourceManager) Fetch(ctx context.Context, name string) ([]domain.Channel, []domain.Program, error) {
	m.mu.RLock()
	src, ok := m.sources[name]
	m.mu.RUnlock()

	if !ok {
		return nil, nil, fmt.Errorf("source not found: %s", name)
	}

	data, err := m.fetchRaw(ctx, src)
	if err != nil {
		return nil, nil, err
	}

	channels, programs, err := xmltv.Parse(data, name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse XMLTV from %s: %w", name, err)
	}

	return channels, programs, nil
}

func (m *ExternalSourceManager) fetchRaw(ctx context.Context, src ExternalSource) ([]byte, error) {
	localPath := ""
	if strings.HasPrefix(src.URL, "file://") {
		localPath = strings.TrimPrefix(src.URL, "file://")
	} else if strings.HasPrefix(src.URL, "/") || (len(src.URL) > 2 && src.URL[1] == ':') {
		localPath = src.URL
	}

	var reader io.Reader
	if localPath != "" {
		f, err := os.Open(localPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", localPath, err)
		}
		defer f.Close()
		reader = f
	} else {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to build request for %s: %w", src.URL, err)
		}
		resp, err := m.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch %s: %w", src.URL, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, src.URL)
		}
		reader = resp.Body
	}

	if src.Type == "xmltv-gz" || strings.HasSuffix(src.URL, ".gz") {
		gzReader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress gzip: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}
	return data, nil
}

func (m *ExternalSourceManager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.path == "" {
		return nil
	}

	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".sources-*.gob.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if err := gob.NewEncoder(tmp).Encode(externalSnapshot{Sources: m.sources}); err != nil {
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

	return os.Rename(tmpPath, m.path)
}

func (m *ExternalSourceManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.path == "" {
		return nil
	}

	f, err := os.Open(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	var snap externalSnapshot
	if err := gob.NewDecoder(f).Decode(&snap); err != nil {
		return err
	}

	m.sources = snap.Sources
	return nil
}
