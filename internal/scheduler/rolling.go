package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gavinmcnair/epg/internal/grabber"
	"github.com/gavinmcnair/epg/internal/plugin"
	"github.com/gavinmcnair/epg/internal/store"
)

const refreshThreshold = 20 * time.Hour

type RollingGrabber struct {
	grabber     *grabber.Grabber
	store       *store.Store
	mu          sync.Mutex
	running     bool
	stopCh      chan struct{}
	doneCh      chan struct{}
	maxParallel int
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewRollingGrabber(g *grabber.Grabber, s *store.Store, maxParallel int) *RollingGrabber {
	if maxParallel < 1 {
		maxParallel = 2
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &RollingGrabber{
		grabber:     g,
		store:       s,
		maxParallel: maxParallel,
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (r *RollingGrabber) Start() {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return
	}
	r.running = true
	r.stopCh = make(chan struct{})
	r.doneCh = make(chan struct{})
	r.mu.Unlock()

	go r.loop()
	log.Println("rolling grabber started")
}

func (r *RollingGrabber) Stop() {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return
	}
	r.cancel()
	close(r.stopCh)
	r.running = false
	doneCh := r.doneCh
	r.mu.Unlock()

	<-doneCh
	log.Println("rolling grabber stopped")
}

func (r *RollingGrabber) loop() {
	defer close(r.doneCh)
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	r.checkAndGrab(r.ctx)

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.checkAndGrab(r.ctx)
		}
	}
}

func (r *RollingGrabber) checkAndGrab(ctx context.Context) {
	grabStats := make(map[string]time.Time)
	for _, gs := range r.store.GetGrabStats() {
		grabStats[gs.Source] = gs.LastGrab
	}

	now := time.Now()
	var toGrab []string

	for _, p := range plugin.List() {
		name := p.Name()

		tz := plugin.Timezone(p)
		loc, err := time.LoadLocation(tz)
		if err != nil {
			loc = time.UTC
		}

		localNow := now.In(loc)
		startOfToday := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, loc)

		lastGrab, hasStats := grabStats[name]
		if !hasStats || lastGrab.Before(startOfToday) {
			toGrab = append(toGrab, name)
		}
	}

	if len(toGrab) == 0 {
		return
	}

	log.Printf("rolling grab: %d sources due", len(toGrab))

	sem := make(chan struct{}, r.maxParallel)
	var wg sync.WaitGroup

	for _, name := range toGrab {
		wg.Add(1)
		go func(source string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			if ctx.Err() != nil {
				return
			}

			log.Printf("rolling grab: starting %s", source)
			results, err := r.grabber.GrabSource(ctx, source)
			if err != nil {
				log.Printf("rolling grab: %s failed: %v", source, err)
				return
			}

			total := 0
			errors := 0
			channelSet := make(map[string]bool)
			totalChannelSet := make(map[string]bool)
			for _, result := range results {
				total += result.Count
				if result.Err != nil {
					errors++
				}
				if result.Channel.ID != "" {
					totalChannelSet[result.Channel.ID] = true
				}
				if result.Count > 0 {
					channelSet[result.Channel.ID] = true
				}
			}

			r.store.RecordGrabStats(source, len(totalChannelSet), len(channelSet), total, errors, 0)
			r.store.SaveSource(source)
			log.Printf("rolling grab: %s done (%d programs, %d errors)", source, total, errors)
		}(name)
	}

	wg.Wait()
}
