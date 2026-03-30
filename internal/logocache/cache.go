package logocache

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
)

type Cache struct {
	dir    string
	mu     sync.RWMutex
	index  map[string]string // safeBase -> source URL
	files  map[string]string // safeBase -> filename on disk
	client *http.Client
}

func New(dir string) *Cache {
	c := &Cache{
		dir:    dir,
		index:  make(map[string]string),
		files:  make(map[string]string),
		client: &http.Client{Timeout: 30 * time.Second},
	}
	if dir != "" {
		os.MkdirAll(dir, 0755)
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
		key := strings.TrimSuffix(name, filepath.Ext(name))
		c.files[key] = name
	}
	log.Printf("logo cache: found %d cached logos in %s", len(c.files), c.dir)
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
	key := safeBase(xmltvID)

	c.mu.RLock()
	filename := c.files[key]
	sourceURL := c.index[key]
	c.mu.RUnlock()

	if filename != "" && c.dir != "" {
		http.ServeFile(w, r, filepath.Join(c.dir, filename))
		return
	}

	if sourceURL == "" {
		http.NotFound(w, r)
		return
	}

	filename, err := c.download(key, sourceURL)
	if err != nil {
		log.Printf("logo cache: failed to fetch %s: %v", sourceURL, err)
		http.Redirect(w, r, sourceURL, http.StatusFound)
		return
	}

	if c.dir != "" {
		http.ServeFile(w, r, filepath.Join(c.dir, filename))
	} else {
		http.Redirect(w, r, sourceURL, http.StatusFound)
	}
}

func (c *Cache) download(key, sourceURL string) (string, error) {
	resp, err := c.client.Get(sourceURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ext := extFromContentType(resp.Header.Get("Content-Type"))
	if ext == "" {
		ext = urlExt(sourceURL)
	}
	if ext == "" {
		ext = ".png"
	}

	filename := key + ext

	if c.dir == "" {
		c.mu.Lock()
		c.files[key] = filename
		c.mu.Unlock()
		return filename, nil
	}

	tmp := filepath.Join(c.dir, filename+".tmp")
	out, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		os.Remove(tmp)
		return "", err
	}
	out.Close()

	dest := filepath.Join(c.dir, filename)
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return "", err
	}

	c.mu.Lock()
	c.files[key] = filename
	c.mu.Unlock()

	return filename, nil
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
