package fetchcache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTest(maxEntries int, maxBytes int64) (*Cache[string], *time.Time) {
	now := time.Now()
	c := New(Options[string]{
		TTL: func(v string, err error) time.Duration {
			if err != nil {
				if err.Error() == "short" {
					return time.Minute
				}
				return 0
			}
			return 10 * time.Minute
		},
		Size:       func(v string) int64 { return int64(len(v)) },
		MaxEntries: maxEntries,
		MaxBytes:   maxBytes,
	})
	c.now = func() time.Time { return now }
	return c, &now
}

func TestSingleFlight(t *testing.T) {
	c, _ := newTest(0, 0)
	var loads atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := c.Get("k", func() (string, error) {
				loads.Add(1)
				time.Sleep(30 * time.Millisecond)
				return "v", nil
			})
			if err != nil || v != "v" {
				t.Errorf("got %q %v", v, err)
			}
		}()
	}
	wg.Wait()
	if loads.Load() != 1 {
		t.Fatalf("expected 1 load, got %d", loads.Load())
	}
}

func TestExpiry(t *testing.T) {
	c, now := newTest(0, 0)
	n := 0
	load := func() (string, error) { n++; return "v", nil }
	c.Get("k", load)
	c.Get("k", load)
	if n != 1 {
		t.Fatalf("loads %d", n)
	}
	*now = now.Add(11 * time.Minute)
	c.Get("k", load)
	if n != 2 {
		t.Fatalf("expected reload, loads %d", n)
	}
}

func TestZeroTTLSharedWithWaitersThenDropped(t *testing.T) {
	c, _ := newTest(0, 0)
	var loads atomic.Int32
	release := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Get("k", func() (string, error) {
				loads.Add(1)
				<-release
				return "", errors.New("boom")
			})
			if err == nil {
				t.Error("expected error")
			}
		}()
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	if loads.Load() != 1 {
		t.Fatalf("waiters should share the failure, loads %d", loads.Load())
	}
	c.Get("k", func() (string, error) { loads.Add(1); return "v", nil })
	if loads.Load() != 2 {
		t.Fatal("a zero-TTL failure must not be retained")
	}
}

func TestShortErrorTTLRetained(t *testing.T) {
	c, now := newTest(0, 0)
	n := 0
	load := func() (string, error) { n++; return "", errors.New("short") }
	c.Get("k", load)
	c.Get("k", load)
	if n != 1 {
		t.Fatalf("loads %d", n)
	}
	*now = now.Add(2 * time.Minute)
	c.Get("k", load)
	if n != 2 {
		t.Fatalf("loads %d", n)
	}
}

func TestMaxEntries(t *testing.T) {
	c, now := newTest(2, 0)
	for _, k := range []string{"a", "b", "c", "d"} {
		*now = now.Add(time.Second)
		c.Get(k, func() (string, error) { return k, nil })
	}
	if len(c.items) != 2 {
		t.Fatalf("expected 2 entries, have %d", len(c.items))
	}
	if _, ok := c.items["d"]; !ok {
		t.Fatal("newest entry must be kept")
	}
}

func TestMaxBytesKeepsNewestEvenIfOversized(t *testing.T) {
	c, now := newTest(0, 10)
	*now = now.Add(time.Second)
	c.Get("a", func() (string, error) { return "12345", nil })
	*now = now.Add(time.Second)
	c.Get("b", func() (string, error) { return "123456789012345", nil })
	if _, ok := c.items["b"]; !ok {
		t.Fatal("oversized newest entry should still be served from cache")
	}
	if _, ok := c.items["a"]; ok {
		t.Fatal("older entry should be evicted to honour the byte budget")
	}
}

func TestFailuresDoNotCountTowardBounds(t *testing.T) {
	c, now := newTest(1, 0)
	for i := 0; i < 5; i++ {
		*now = now.Add(time.Second)
		k := string(rune('a' + i))
		c.Get(k, func() (string, error) { return "", errors.New("short") })
	}
	*now = now.Add(time.Second)
	c.Get("good", func() (string, error) { return "v", nil })
	if _, ok := c.items["good"]; !ok {
		t.Fatal("success entry evicted by cached failures")
	}
}

func TestDocTTL(t *testing.T) {
	ttl := DocTTL[string](15 * time.Minute)
	if ttl("v", nil) != 15*time.Minute {
		t.Fatal("success ttl")
	}
	if ttl("", &StatusError{Code: 404, Source: "x"}) != time.Hour {
		t.Fatal("404 ttl")
	}
	if ttl("", &StatusError{Code: 410, Source: "x"}) != time.Hour {
		t.Fatal("410 ttl")
	}
	if ttl("", &StatusError{Code: 500, Source: "x"}) != 2*time.Minute {
		t.Fatal("500 ttl")
	}
	if ttl("", errors.New("net")) != 2*time.Minute {
		t.Fatal("generic error ttl")
	}
	if (&StatusError{Code: 404, Source: "epgshare"}).Error() != "HTTP 404 from epgshare" {
		t.Fatal("error text must match the old message")
	}
}
