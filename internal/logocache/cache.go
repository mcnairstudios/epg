package logocache

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Cache fetches a manifest from a remote logo store and rewrites
// channel logo URLs to point at that store. No local disk I/O.
type Cache struct {
	manifestURL string
	baseURL     string
	mu          sync.RWMutex
	index       map[string]string // safe base xmltv_id -> filename
	client      *http.Client
}

func New(manifestURL, baseURL string) *Cache {
	return &Cache{
		manifestURL: manifestURL,
		baseURL:     baseURL,
		index:       make(map[string]string),
		client:      &http.Client{Timeout: 15 * time.Second},
	}
}

// Load fetches the manifest from the remote logo store.
func (c *Cache) Load() error {
	if c.manifestURL == "" {
		return nil
	}
	resp, err := c.client.Get(c.manifestURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var m map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return err
	}
	c.mu.Lock()
	c.index = m
	c.mu.Unlock()
	log.Printf("logo cache: loaded %d logos from manifest", len(m))
	return nil
}

// Has reports whether a cached logo exists for the given xmltv_id.
func (c *Cache) Has(xmltvID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.index[safeBase(xmltvID)]
	return ok
}

// URL returns the remote URL for the logo, or "" if not cached.
func (c *Cache) URL(xmltvID string) string {
	c.mu.RLock()
	filename := c.index[safeBase(xmltvID)]
	c.mu.RUnlock()
	if filename == "" || c.baseURL == "" {
		return ""
	}
	return c.baseURL + "/" + filename
}

// Count returns the number of logos in the manifest.
func (c *Cache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.index)
}

// safeBase strips the @quality suffix and sanitises for use as a filename/URL key.
func safeBase(xmltvID string) string {
	id := xmltvID
	if i := strings.Index(id, "@"); i != -1 {
		id = id[:i]
	}
	r := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return r.Replace(id)
}

// SafeBase is exported for use by the downloader.
func SafeBase(xmltvID string) string { return safeBase(xmltvID) }
