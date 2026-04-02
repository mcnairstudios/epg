package grabber

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
	"github.com/gavinmcnair/epg/internal/store"
)

type Grabber struct {
	store          *store.Store
	maxConnections int
	days           int
	progress       *ProgressTracker
	grabbingMu     sync.Mutex
	grabbing       map[string]bool
	log            zerolog.Logger
}

func New(s *store.Store, maxConnections, days int, logger zerolog.Logger) *Grabber {
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
		progress:       NewProgressTracker(),
		grabbing:       make(map[string]bool),
		log:            logger,
	}
}

func (g *Grabber) Progress() *ProgressTracker {
	return g.progress
}

func (g *Grabber) tryLock(source string) bool {
	g.grabbingMu.Lock()
	defer g.grabbingMu.Unlock()
	if g.grabbing[source] {
		return false
	}
	g.grabbing[source] = true
	return true
}

func (g *Grabber) unlock(source string) {
	g.grabbingMu.Lock()
	defer g.grabbingMu.Unlock()
	delete(g.grabbing, source)
}

func (g *Grabber) IsGrabbing(source string) bool {
	g.grabbingMu.Lock()
	defer g.grabbingMu.Unlock()
	return g.grabbing[source]
}

type Result struct {
	Source  string
	Channel domain.Channel
	Date    time.Time
	Count   int
	Err     error
}

type Summary struct {
	TotalChannels    int
	ChannelsWithData int
	Programs         int
	Errors           int
}

func Summarize(results []Result) Summary {
	var s Summary
	channelSet := make(map[string]bool)
	totalChannelSet := make(map[string]bool)
	for _, r := range results {
		s.Programs += r.Count
		if r.Err != nil {
			s.Errors++
		}
		if r.Channel.ID != "" {
			totalChannelSet[r.Channel.ID] = true
		}
		if r.Count > 0 {
			channelSet[r.Channel.ID] = true
		}
	}
	s.TotalChannels = len(totalChannelSet)
	s.ChannelsWithData = len(channelSet)
	return s
}

func (g *Grabber) GrabSource(ctx context.Context, sourceName string) ([]Result, error) {
	p, ok := plugin.Get(sourceName)
	if !ok {
		return nil, fmt.Errorf("unknown source: %s", sourceName)
	}

	if !g.tryLock(sourceName) {
		return nil, fmt.Errorf("source %s is already being grabbed", sourceName)
	}
	defer g.unlock(sourceName)

	channels, err := p.Channels()
	if err != nil {
		return nil, fmt.Errorf("failed to get channels for %s: %w", sourceName, err)
	}

	g.store.SetChannels(channels)

	return g.grabChannels(ctx, p, channels), nil
}

func (g *Grabber) GrabAll(ctx context.Context, skip map[string]bool) []Result {
	var allResults []Result

	for _, p := range plugin.List() {
		if skip[p.Name()] {
			continue
		}

		select {
		case <-ctx.Done():
			return allResults
		default:
		}

		if !g.tryLock(p.Name()) {
			continue
		}

		channels, err := p.Channels()
		if err != nil {
			g.unlock(p.Name())
			allResults = append(allResults, Result{
				Source: p.Name(),
				Err:    err,
			})
			continue
		}

		g.store.SetChannels(channels)
		results := g.grabChannels(ctx, p, channels)
		g.unlock(p.Name())
		allResults = append(allResults, results...)
	}

	return allResults
}

func (g *Grabber) grabChannels(ctx context.Context, p plugin.Provider, channels []domain.Channel) []Result {
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

	g.progress.Start(p.Name(), len(jobs))

	results := make([]Result, len(jobs))
	sem := make(chan struct{}, g.maxConnections)
	var wg sync.WaitGroup

	for i, j := range jobs {
		wg.Add(1)
		go func(idx int, j job) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				results[idx] = Result{Source: p.Name(), Channel: j.channel, Date: j.date, Err: ctx.Err()}
				g.progress.Record(p.Name(), true)
				return
			}
			defer func() { <-sem }()

			if ctx.Err() != nil {
				results[idx] = Result{Source: p.Name(), Channel: j.channel, Date: j.date, Err: ctx.Err()}
				g.progress.Record(p.Name(), true)
				return
			}

			programs, err := p.Fetch(j.channel, j.date)
			results[idx] = Result{
				Source:  p.Name(),
				Channel: j.channel,
				Date:    j.date,
				Err:     err,
			}

			if err != nil {
				g.log.Error().Err(err).Str("source", p.Name()).Str("channel", j.channel.ID).Str("date", j.date.Format("2006-01-02")).Msg("fetch error")
			} else if len(programs) > 0 {
				results[idx].Count = len(programs)
				g.store.SetPrograms(j.channel.ID, programs)
			}

			g.progress.Record(p.Name(), err != nil)
		}(i, j)
	}

	wg.Wait()
	g.progress.Finish(p.Name())
	return results
}
