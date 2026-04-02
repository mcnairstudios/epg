package scheduler_test

import (
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/grabber"
	"github.com/gavinmcnair/epg/internal/plugin"
	"github.com/gavinmcnair/epg/internal/scheduler"
	"github.com/gavinmcnair/epg/internal/store"
)

type rollingTestProvider struct{}

func (p *rollingTestProvider) Name() string        { return "rolling-test" }
func (p *rollingTestProvider) Description() string { return "rolling test provider" }
func (p *rollingTestProvider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "r1", Name: "Rolling Ch", Source: "rolling-test", XmltvID: "Rolling.test"},
	}, nil
}
func (p *rollingTestProvider) Fetch(ch domain.Channel, date time.Time) ([]domain.Program, error) {
	return nil, nil
}

func TestRollingGrabberNoStatsTriggersGrab(t *testing.T) {
	plugin.Reset()
	plugin.Register(&rollingTestProvider{})

	s := store.New("")
	g := grabber.New(s, 1, 1, zerolog.Nop())
	rg := scheduler.NewRollingGrabber(g, s, 1, zerolog.Nop())

	if stats := s.GetGrabStats(); len(stats) != 0 {
		t.Fatalf("expected no grab stats initially, got %d", len(stats))
	}

	rg.Start()
	time.Sleep(500 * time.Millisecond)
	rg.Stop()

	stats := s.GetGrabStats()
	if len(stats) == 0 {
		t.Fatal("expected grab stats to be recorded after rolling grab with no prior stats")
	}

	found := false
	for _, gs := range stats {
		if gs.Source == "rolling-test" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected grab stats for rolling-test source")
	}
}

func TestRollingGrabberGrabbedTodaySkipped(t *testing.T) {
	plugin.Reset()
	plugin.Register(&rollingTestProvider{})

	s := store.New("")
	g := grabber.New(s, 1, 1, zerolog.Nop())

	s.RecordGrabStats("rolling-test", 1, 1, 10, 0, time.Second)

	var grabBefore time.Time
	for _, gs := range s.GetGrabStats() {
		if gs.Source == "rolling-test" {
			grabBefore = gs.LastGrab
			break
		}
	}

	if grabBefore.IsZero() {
		t.Fatal("expected grab stats to be set before run")
	}

	rg := scheduler.NewRollingGrabber(g, s, 1, zerolog.Nop())
	rg.Start()
	time.Sleep(500 * time.Millisecond)
	rg.Stop()

	var grabAfter time.Time
	for _, gs := range s.GetGrabStats() {
		if gs.Source == "rolling-test" {
			grabAfter = gs.LastGrab
			break
		}
	}

	if !grabAfter.Equal(grabBefore) {
		t.Fatalf("expected grab to be skipped for source grabbed today: before=%v after=%v", grabBefore, grabAfter)
	}
}

func TestRollingGrabberGrabbedYesterdayRescheduled(t *testing.T) {
	plugin.Reset()
	plugin.Register(&rollingTestProvider{})

	s := store.New("")
	g := grabber.New(s, 1, 1, zerolog.Nop())

	yesterday := time.Now().UTC().Add(-25 * time.Hour)
	loc := time.UTC
	startOfToday := time.Date(time.Now().UTC().Year(), time.Now().UTC().Month(), time.Now().UTC().Day(), 0, 0, 0, 0, loc)
	if !yesterday.Before(startOfToday) {
		t.Skip("test requires yesterday to be before start of today")
	}

	s.SetGrabStatsForTest("rolling-test", store.GrabStats{
		Source:   "rolling-test",
		LastGrab: yesterday,
	})

	stats := s.GetGrabStats()
	found := false
	for _, gs := range stats {
		if gs.Source == "rolling-test" {
			found = true
			if !gs.LastGrab.Before(startOfToday) {
				t.Fatalf("expected LastGrab to be before today, got %v", gs.LastGrab)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected grab stats to be pre-set")
	}

	rg := scheduler.NewRollingGrabber(g, s, 1, zerolog.Nop())
	rg.Start()
	time.Sleep(500 * time.Millisecond)
	rg.Stop()

	var grabAfter time.Time
	for _, gs := range s.GetGrabStats() {
		if gs.Source == "rolling-test" {
			grabAfter = gs.LastGrab
			break
		}
	}

	if grabAfter.Before(startOfToday) {
		t.Fatalf("expected source grabbed yesterday to be rescheduled and have LastGrab updated, got %v", grabAfter)
	}
}
