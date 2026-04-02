package scheduler

import (
	"context"
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"

	"github.com/gavinmcnair/epg/internal/grabber"
	"github.com/gavinmcnair/epg/internal/store"
)

type Scheduler struct {
	cron     *cron.Cron
	grabber  *grabber.Grabber
	store    *store.Store
	disabled *store.DisabledStore
	mu       sync.Mutex
	running  bool
	entryID  cron.EntryID
	ctx      context.Context
	cancel   context.CancelFunc
	log      zerolog.Logger
}

func New(g *grabber.Grabber, s *store.Store, disabled *store.DisabledStore, logger zerolog.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		cron:     cron.New(),
		grabber:  g,
		store:    s,
		disabled: disabled,
		ctx:      ctx,
		cancel:   cancel,
		log:      logger,
	}
}

func (s *Scheduler) Start(schedule string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		s.cron.Remove(s.entryID)
	}

	id, err := s.cron.AddFunc(schedule, func() {
		s.runGrabAll()
	})
	if err != nil {
		return err
	}

	s.entryID = id
	s.running = true
	s.cron.Start()
	s.log.Info().Str("schedule", schedule).Msg("scheduler started")
	return nil
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		s.cancel()
		<-s.cron.Stop().Done()
		s.running = false
		s.log.Info().Msg("scheduler stopped")
	}
}

func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Scheduler) runGrabAll() {
	s.log.Info().Msg("scheduled grab starting")
	results := s.grabber.GrabAll(s.ctx, s.disabled.List())

	total := 0
	errors := 0
	for _, r := range results {
		total += r.Count
		if r.Err != nil {
			errors++
		}
	}

	if err := s.store.Save(); err != nil {
		s.log.Error().Err(err).Msg("scheduled grab: failed to save")
	}

	s.log.Info().Int("programs", total).Int("errors", errors).Msg("scheduled grab complete")
}
