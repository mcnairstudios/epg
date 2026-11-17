package epgshare

import (
	"fmt"
	"sync"
	"time"
)

const (
	cacheTTL      = 15 * time.Minute
	notFoundTTL   = time.Hour
	errorTTL      = 2 * time.Minute
	maxCacheItems = 4
)

type statusError struct{ code int }

func (e *statusError) Error() string { return fmt.Sprintf("HTTP %d from epgshare", e.code) }

type cacheItem struct {
	ready   chan struct{}
	idx     index
	err     error
	expires time.Time
	created time.Time
}

type tagCache struct {
	mu    sync.Mutex
	items map[string]*cacheItem
	now   func() time.Time
}

func newTagCache() *tagCache {
	return &tagCache{items: make(map[string]*cacheItem), now: time.Now}
}

func (c *tagCache) get(tag string, load func(string) (index, error)) (index, error) {
	c.mu.Lock()
	it, ok := c.items[tag]
	if ok && it.isDone() && !c.now().Before(it.expires) {
		delete(c.items, tag)
		ok = false
	}
	if ok {
		c.mu.Unlock()
		<-it.ready
		return it.idx, it.err
	}

	it = &cacheItem{ready: make(chan struct{}), created: c.now()}
	c.items[tag] = it
	c.evictLocked()
	c.mu.Unlock()

	idx, err := load(tag)

	c.mu.Lock()
	it.idx, it.err = idx, err
	it.expires = c.now().Add(ttlFor(err))
	c.mu.Unlock()
	close(it.ready)
	return idx, err
}

func (it *cacheItem) isDone() bool {
	select {
	case <-it.ready:
		return true
	default:
		return false
	}
}

func ttlFor(err error) time.Duration {
	if err == nil {
		return cacheTTL
	}
	if se, ok := err.(*statusError); ok && se.code == 404 {
		return notFoundTTL
	}
	return errorTTL
}

func (c *tagCache) evictLocked() {
	for {
		live := 0
		var oldestTag string
		var oldest *cacheItem
		for tag, it := range c.items {
			done := it.isDone()
			if done && it.err != nil {
				continue
			}
			live++
			if done && (oldest == nil || it.created.Before(oldest.created)) {
				oldestTag, oldest = tag, it
			}
		}
		if live <= maxCacheItems || oldest == nil {
			return
		}
		delete(c.items, oldestTag)
	}
}
