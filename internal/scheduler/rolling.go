package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/gavinmcnair/epg/internal/grabber"
	"github.com/gavinmcnair/epg/internal/plugin"
	"github.com/gavinmcnair/epg/internal/store"
)

const refreshThreshold = 20 * time.Hour

type RollingGrabber struct {
	grabber     *grabber.Grabber
	store       *store.Store
	disabled    *store.DisabledStore
	mu          sync.Mutex
	running     bool
	stopCh      chan struct{}
	doneCh      chan struct{}
	maxParallel int
	ctx         context.Context
	cancel      context.CancelFunc
	log         zerolog.Logger
}

func NewRollingGrabber(g *grabber.Grabber, s *store.Store, disabled *store.DisabledStore, maxParallel int, logger zerolog.Logger) *RollingGrabber {
	if maxParallel < 1 {
		maxParallel = 2
	}
	return &RollingGrabber{
		grabber:     g,
		store:       s,
		disabled:    disabled,
		maxParallel: maxParallel,
		log:         logger,
	}
}

func (r *RollingGrabber) Start() {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.ctx = ctx
	r.cancel = cancel
	r.running = true
	r.stopCh = make(chan struct{})
	r.doneCh = make(chan struct{})
	r.mu.Unlock()

	go r.loop()
	r.log.Info().Msg("rolling grabber started")
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
	r.log.Info().Msg("rolling grabber stopped")
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
	disabled := r.disabled.List()

	grabStats := make(map[string]time.Time)
	for _, gs := range r.store.GetGrabStats() {
		grabStats[gs.Source] = gs.LastGrab
	}

	now := time.Now()
	var toGrab []string

	for _, p := range plugin.List() {
		name := p.Name()
		if disabled[name] {
			continue
		}

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

	r.log.Info().Int("count", len(toGrab)).Msg("rolling grab: sources due")

	work := make(chan string, len(toGrab))
	for _, name := range toGrab {
		work <- name
	}
	close(work)

	var wg sync.WaitGroup
	for i := 0; i < r.maxParallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for source := range work {
				if ctx.Err() != nil {
					return
				}

				r.log.Info().Str("source", source).Msg("rolling grab starting")
				results, err := r.grabber.GrabSource(ctx, source)
				if err != nil {
					r.log.Error().Err(err).Str("source", source).Msg("rolling grab failed")
					continue
				}

				summary := grabber.Summarize(results)
				r.store.RecordGrabStats(source, summary.TotalChannels, summary.ChannelsWithData, summary.Programs, summary.Errors, 0)
				if err := r.store.SaveSource(source); err != nil {
					r.log.Error().Err(err).Str("source", source).Msg("failed to persist source data")
				}
				r.log.Info().Str("source", source).Int("programs", summary.Programs).Int("errors", summary.Errors).Msg("rolling grab done")
			}
		}()
	}

	wg.Wait()
}
