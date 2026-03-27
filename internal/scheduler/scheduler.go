package scheduler

import (
	"log"
	"sync"

	"github.com/robfig/cron/v3"

	"github.com/gavinmcnair/epg/internal/grabber"
	"github.com/gavinmcnair/epg/internal/store"
)

type Scheduler struct {
	cron    *cron.Cron
	grabber *grabber.Grabber
	store   *store.Store
	mu      sync.Mutex
	running bool
	entryID cron.EntryID
}

func New(g *grabber.Grabber, s *store.Store) *Scheduler {
	return &Scheduler{
		cron:    cron.New(),
		grabber: g,
		store:   s,
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
	log.Printf("scheduler started with schedule: %s", schedule)
	return nil
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		s.cron.Stop()
		s.running = false
		log.Println("scheduler stopped")
	}
}

func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Scheduler) runGrabAll() {
	log.Println("scheduled grab starting...")
	results := s.grabber.GrabAll()

	total := 0
	errors := 0
	for _, r := range results {
		total += r.Count
		if r.Err != nil {
			errors++
		}
	}

	if err := s.store.Save(); err != nil {
		log.Printf("scheduled grab: failed to save: %v", err)
	}

	log.Printf("scheduled grab complete: %d programs, %d errors", total, errors)
}
