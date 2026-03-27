package grabber

import (
	"fmt"
	"sync"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
	"github.com/gavinmcnair/epg/internal/store"
)

type Grabber struct {
	store          *store.Store
	maxConnections int
	days           int
}

func New(s *store.Store, maxConnections, days int) *Grabber {
	if maxConnections < 1 {
		maxConnections = 1
	}
	if days < 1 {
		days = 1
	}
	return &Grabber{
		store:          s,
		maxConnections: maxConnections,
		days:           days,
	}
}

type Result struct {
	Source   string
	Channel domain.Channel
	Date    time.Time
	Count   int
	Err     error
}

func (g *Grabber) GrabSource(sourceName string) ([]Result, error) {
	p, ok := plugin.Get(sourceName)
	if !ok {
		return nil, fmt.Errorf("unknown source: %s", sourceName)
	}

	channels, err := p.Channels()
	if err != nil {
		return nil, fmt.Errorf("failed to get channels for %s: %w", sourceName, err)
	}

	g.store.SetChannels(channels)

	return g.grabChannels(p, channels), nil
}

func (g *Grabber) GrabAll() []Result {
	var allResults []Result

	for _, p := range plugin.List() {
		channels, err := p.Channels()
		if err != nil {
			allResults = append(allResults, Result{
				Source: p.Name(),
				Err:    err,
			})
			continue
		}

		g.store.SetChannels(channels)
		results := g.grabChannels(p, channels)
		allResults = append(allResults, results...)
	}

	return allResults
}

func (g *Grabber) grabChannels(p plugin.Provider, channels []domain.Channel) []Result {
	type job struct {
		channel domain.Channel
		date    time.Time
	}

	var jobs []job
	now := time.Now().UTC().Truncate(24 * time.Hour)
	for _, ch := range channels {
		g.store.ClearPrograms(ch.ID)
		for d := 0; d < g.days; d++ {
			jobs = append(jobs, job{channel: ch, date: now.Add(time.Duration(d) * 24 * time.Hour)})
		}
	}

	results := make([]Result, len(jobs))
	sem := make(chan struct{}, g.maxConnections)
	var wg sync.WaitGroup

	for i, j := range jobs {
		wg.Add(1)
		go func(idx int, j job) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			programs, err := p.Fetch(j.channel, j.date)
			results[idx] = Result{
				Source:  p.Name(),
				Channel: j.channel,
				Date:    j.date,
				Err:     err,
			}

			if err == nil && len(programs) > 0 {
				results[idx].Count = len(programs)
				g.store.SetPrograms(j.channel.ID, programs)
			}
		}(i, j)
	}

	wg.Wait()
	return results
}
