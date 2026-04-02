package grabber

import (
	"sync"
	"time"
)

type SourceProgress struct {
	Source  string `json:"source"`
	Total   int    `json:"total"`
	Done    int    `json:"done"`
	Errors  int    `json:"errors"`
	Active  bool   `json:"active"`
}

type ProgressTracker struct {
	mu      sync.Mutex
	sources map[string]*SourceProgress
	done    chan struct{}
}

func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		sources: make(map[string]*SourceProgress),
		done:    make(chan struct{}),
	}
}

func (t *ProgressTracker) Close() {
	close(t.done)
}

func (t *ProgressTracker) Start(source string, total int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sources[source] = &SourceProgress{
		Source: source,
		Total:  total,
		Active: true,
	}
}

func (t *ProgressTracker) Record(source string, isError bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	p, ok := t.sources[source]
	if !ok {
		return
	}
	p.Done++
	if isError {
		p.Errors++
	}
}

func (t *ProgressTracker) Finish(source string) {
	t.mu.Lock()
	p, ok := t.sources[source]
	if ok {
		p.Active = false
	}
	t.mu.Unlock()

	go func() {
		select {
		case <-time.After(30 * time.Second):
			t.mu.Lock()
			delete(t.sources, source)
			t.mu.Unlock()
		case <-t.done:
		}
	}()
}

func (t *ProgressTracker) All() map[string]SourceProgress {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := make(map[string]SourceProgress, len(t.sources))
	for k, v := range t.sources {
		result[k] = *v
	}
	return result
}
