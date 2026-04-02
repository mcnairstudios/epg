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

type fakeProvider struct{}

func (f *fakeProvider) Name() string        { return "sched-test" }
func (f *fakeProvider) Description() string { return "test" }
func (f *fakeProvider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{{ID: "ch1", Name: "Test", Source: "sched-test", XmltvID: "Test.sched"}}, nil
}
func (f *fakeProvider) Fetch(ch domain.Channel, date time.Time) ([]domain.Program, error) {
	return []domain.Program{{Channel: ch.XmltvID, Title: "Show", Start: date, Stop: date.Add(time.Hour)}}, nil
}

func TestSchedulerStartStop(t *testing.T) {
	plugin.Reset()
	plugin.Register(&fakeProvider{})

	s := store.New("")
	g := grabber.New(s, 1, 1, zerolog.Nop())
	sched := scheduler.New(g, s, store.NewDisabledStore(""), zerolog.Nop())

	if sched.IsRunning() {
		t.Fatal("scheduler should not be running initially")
	}

	if err := sched.Start("@every 1h"); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	if !sched.IsRunning() {
		t.Fatal("scheduler should be running after start")
	}

	sched.Stop()

	if sched.IsRunning() {
		t.Fatal("scheduler should not be running after stop")
	}
}

func TestSchedulerInvalidCron(t *testing.T) {
	plugin.Reset()

	s := store.New("")
	g := grabber.New(s, 1, 1, zerolog.Nop())
	sched := scheduler.New(g, s, store.NewDisabledStore(""), zerolog.Nop())

	if err := sched.Start("invalid cron expression"); err == nil {
		t.Fatal("expected error on invalid cron expression")
	}
}
