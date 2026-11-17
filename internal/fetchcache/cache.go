// Package fetchcache provides a small single-flight cache with per-result
// expiry and a bound on how much it retains. Plugins use it so that many
// channels needing the same upstream document trigger one download.
package fetchcache

import (
	"sync"
	"time"
)

type Options[V any] struct {
	// TTL returns how long a finished result is retained. Zero means the
	// result is handed to callers already waiting on it and then dropped.
	TTL func(v V, err error) time.Duration
	// Size reports the retained size of a successful value, in bytes.
	Size func(v V) int64
	// MaxEntries bounds the number of successful entries kept (0 = unbounded).
	MaxEntries int
	// MaxBytes bounds the summed Size of successful entries (0 = unbounded).
	MaxBytes int64
}

type item[V any] struct {
	ready   chan struct{}
	val     V
	err     error
	size    int64
	expires time.Time
	created time.Time
}

func (it *item[V]) done() bool {
	select {
	case <-it.ready:
		return true
	default:
		return false
	}
}

type Cache[V any] struct {
	opts  Options[V]
	mu    sync.Mutex
	items map[string]*item[V]
	now   func() time.Time
}

func New[V any](opts Options[V]) *Cache[V] {
	return &Cache[V]{opts: opts, items: make(map[string]*item[V]), now: time.Now}
}

// Get returns the cached value for key, calling load once if it is absent or
// expired. Concurrent callers for the same key wait for that single load.
func (c *Cache[V]) Get(key string, load func() (V, error)) (V, error) {
	c.mu.Lock()
	it, ok := c.items[key]
	if ok && it.done() && !c.now().Before(it.expires) {
		delete(c.items, key)
		ok = false
	}
	if ok {
		c.mu.Unlock()
		<-it.ready
		return it.val, it.err
	}

	it = &item[V]{ready: make(chan struct{}), created: c.now()}
	c.items[key] = it
	c.mu.Unlock()

	val, err := load()

	c.mu.Lock()
	it.val, it.err = val, err
	var ttl time.Duration
	if c.opts.TTL != nil {
		ttl = c.opts.TTL(val, err)
	}
	it.expires = c.now().Add(ttl)
	if ttl <= 0 {
		if c.items[key] == it {
			delete(c.items, key)
		}
	} else if err == nil {
		if c.opts.Size != nil {
			it.size = c.opts.Size(val)
		}
		c.evictLocked(it)
	}
	c.mu.Unlock()
	close(it.ready)
	return val, err
}

// evictLocked drops the oldest finished successful entries until the bounds
// hold. Failures (small, short-lived) and in-flight loads are never evicted,
// and neither is keep, the entry that was just stored.
func (c *Cache[V]) evictLocked(keep *item[V]) {
	for {
		var live int
		var bytes int64
		var oldestKey string
		var oldest *item[V]
		for k, it := range c.items {
			if !it.done() && it != keep {
				continue
			}
			if it.err != nil {
				continue
			}
			live++
			bytes += it.size
			if it != keep && (oldest == nil || it.created.Before(oldest.created)) {
				oldestKey, oldest = k, it
			}
		}
		overEntries := c.opts.MaxEntries > 0 && live > c.opts.MaxEntries
		overBytes := c.opts.MaxBytes > 0 && bytes > c.opts.MaxBytes
		if (!overEntries && !overBytes) || oldest == nil {
			return
		}
		delete(c.items, oldestKey)
	}
}
