package bandwidth

import (
	"sync"
	"sync/atomic"
	"time"
)

var Global = New()

type sample struct {
	t time.Time
	n int64
}

type Tracker struct {
	mu      sync.Mutex
	total   atomic.Int64
	samples []sample
}

func New() *Tracker {
	return &Tracker{}
}

func (t *Tracker) Add(n int64) {
	t.total.Add(n)
	now := time.Now()
	cutoff := now.Add(-5 * time.Second)
	t.mu.Lock()
	t.samples = append(t.samples, sample{now, n})
	for len(t.samples) > 0 && t.samples[0].t.Before(cutoff) {
		t.samples = t.samples[1:]
	}
	t.mu.Unlock()
}

func (t *Tracker) Rate() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	var sum int64
	for _, s := range t.samples {
		sum += s.n
	}
	return float64(sum) / 5.0
}

func (t *Tracker) Total() int64 {
	return t.total.Load()
}
