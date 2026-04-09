package logocache

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/gavinmcnair/epg/internal/domain"
)

//go:embed placeholder_tv.svg
var placeholderTV []byte

//go:embed placeholder_radio.svg
var placeholderRadio []byte

type cached struct {
	filename    string
	contentType string
}

type Cache struct {
	dir    string
	mu     sync.RWMutex
	index  map[string]string
	files  map[string]cached
	client *http.Client
	log    zerolog.Logger
}

func New(dir string, logger zerolog.Logger) *Cache {
	c := &Cache{
		dir:    dir,
		index:  make(map[string]string),
		files:  make(map[string]cached),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: logger,
	}
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			logger.Error().Err(err).Str("dir", dir).Msg("logo cache: failed to create directory")
		}
		c.scanDir()
	}
	return c
}

func (c *Cache) scanDir() {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range entries {
		if e.IsDir() || e.Name() == "manifest.json" {
			continue
		}
		name := e.Name()
		ext := filepath.Ext(name)
		key := strings.TrimSuffix(name, ext)
		c.files[key] = cached{filename: name, contentType: contentTypeForExt(ext)}
	}
	c.log.Info().Int("count", len(c.files)).Str("dir", c.dir).Msg("logo cache: scanned")
}

func (c *Cache) Index(channels []domain.Channel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, ch := range channels {
		if ch.Logo == "" {
			continue
		}
		key := safeBase(ch.XmltvID)
		if key == "" {
			continue
		}
		if _, ok := c.index[key]; !ok {
			c.index[key] = ch.Logo
		}
	}
	c.log.Info().Int("indexed", len(c.index)).Msg("logo cache: indexed channel logos")
}

func (c *Cache) Has(xmltvID string) bool {
	key := safeBase(xmltvID)
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, inFiles := c.files[key]
	_, inIndex := c.index[key]
	return inFiles || inIndex
}

func (c *Cache) Serve(w http.ResponseWriter, r *http.Request, xmltvID string) {
	switch xmltvID {
	case "placeholder-tv":
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(placeholderTV)
		return
	case "placeholder-radio":
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(placeholderRadio)
		return
	}

	key := safeBase(xmltvID)

	c.mu.RLock()
	entry, found := c.files[key]
	sourceURL := c.index[key]
	c.mu.RUnlock()

	if found && c.dir != "" {
		serveFile(w, filepath.Join(c.dir, entry.filename), entry.contentType)
		return
	}

	if sourceURL == "" {
		c.log.Warn().Str("key", key).Str("id", xmltvID).Msg("logo cache: no source URL in index")
		http.NotFound(w, r)
		return
	}

	data, contentType, ext, err := c.fetch(r.Context(), sourceURL)
	if err != nil {
		c.log.Error().Err(err).Str("key", key).Str("url", sourceURL).Msg("logo cache: fetch failed")
		http.NotFound(w, r)
		return
	}

	filename := key + ext
	if c.dir != "" {
		dest := filepath.Join(c.dir, filename)
		if writeErr := os.WriteFile(dest, data, 0644); writeErr != nil {
			c.log.Error().Err(writeErr).Str("dest", dest).Msg("logo cache: write failed")
		} else {
			c.mu.Lock()
			c.files[key] = cached{filename: filename, contentType: contentType}
			c.mu.Unlock()
		}
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(data)
}

func (c *Cache) fetch(ctx context.Context, sourceURL string) ([]byte, string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, "", "", err
	}
	req.Header.Set("User-Agent", "EPG-Logo-Cache/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", "", err
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return nil, "", "", err
	}

	ct := resp.Header.Get("Content-Type")
	ext := extFromContentType(ct)
	if ext == "" {
		ext = urlExt(sourceURL)
	}
	if ext == "" {
		ext = ".png"
	}
	if ct == "" {
		ct = contentTypeForExt(ext)
	}

	return buf.Bytes(), ct, ext, nil
}

func serveFile(w http.ResponseWriter, path, contentType string) {
	data, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(data)
}

func safeBase(xmltvID string) string {
	id := xmltvID
	if i := strings.Index(id, "@"); i != -1 {
		id = id[:i]
	}
	r := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return r.Replace(id)
}

func SafeBase(xmltvID string) string { return safeBase(xmltvID) }

func urlExt(u string) string {
	if idx := strings.LastIndex(u, "?"); idx != -1 {
		u = u[:idx]
	}
	if idx := strings.LastIndex(u, "."); idx != -1 {
		ext := strings.ToLower(u[idx:])
		switch ext {
		case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp":
			return ext
		}
	}
	return ""
}

func extFromContentType(ct string) string {
	ct = strings.Split(ct, ";")[0]
	ct = strings.TrimSpace(ct)
	switch ct {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/svg+xml":
		return ".svg"
	case "image/webp":
		return ".webp"
	}
	return ""
}

func contentTypeForExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	}
	return "image/png"
}
