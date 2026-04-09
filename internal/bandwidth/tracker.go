package bandwidth

import (
	"sync"
	"sync/atomic"
	"time"
)

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
	trim := 0
	for trim < len(t.samples) && t.samples[trim].t.Before(cutoff) {
		trim++
	}
	if trim > 0 {
		copy(t.samples, t.samples[trim:])
		t.samples = t.samples[:len(t.samples)-trim]
	}
	t.mu.Unlock()
}

func (t *Tracker) Rate() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.samples) == 0 {
		return 0
	}
	var sum int64
	for _, s := range t.samples {
		sum += s.n
	}
	window := time.Since(t.samples[0].t).Seconds()
	if window <= 0 {
		return 0
	}
	return float64(sum) / window
}

func (t *Tracker) Total() int64 {
	return t.total.Load()
}
