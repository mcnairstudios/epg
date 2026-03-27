package grabber_test

import (
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/grabber"
	"github.com/gavinmcnair/epg/internal/plugin"
	"github.com/gavinmcnair/epg/internal/store"
)

type fakeProvider struct {
	name     string
	channels []domain.Channel
	programs []domain.Program
	fetchErr error
}

func (f *fakeProvider) Name() string        { return f.name }
func (f *fakeProvider) Description() string { return "fake" }
func (f *fakeProvider) Channels() ([]domain.Channel, error) {
	return f.channels, nil
}
func (f *fakeProvider) Fetch(ch domain.Channel, date time.Time) ([]domain.Program, error) {
	if f.fetchErr != nil {
		return nil, f.fetchErr
	}
	return f.programs, nil
}

func TestGrabSource(t *testing.T) {
	plugin.Reset()

	fp := &fakeProvider{
		name: "test-source",
		channels: []domain.Channel{
			{ID: "ch1", Name: "Test Channel", Source: "test-source", XmltvID: "TestCh.test"},
		},
		programs: []domain.Program{
			{Channel: "TestCh.test", Title: "Test Show", Start: time.Now(), Stop: time.Now().Add(time.Hour)},
		},
	}
	plugin.Register(fp)

	s := store.New("")
	g := grabber.New(s, 1, 1)

	results, err := g.GrabSource("test-source")
	if err != nil {
		t.Fatalf("grab failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	for _, r := range results {
		if r.Err != nil {
			t.Fatalf("result error: %v", r.Err)
		}
	}

	channels := s.GetChannels()
	if len(channels) != 1 {
		t.Fatalf("expected 1 channel in store, got %d", len(channels))
	}

	programs := s.GetPrograms()
	if len(programs) == 0 {
		t.Fatal("expected programs in store")
	}
}

func TestGrabSourceUnknown(t *testing.T) {
	plugin.Reset()

	s := store.New("")
	g := grabber.New(s, 1, 1)

	_, err := g.GrabSource("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown source")
	}
}

func TestGrabAll(t *testing.T) {
	plugin.Reset()

	plugin.Register(&fakeProvider{
		name: "src1",
		channels: []domain.Channel{
			{ID: "ch1", Name: "Ch1", Source: "src1", XmltvID: "Ch1.src1"},
		},
		programs: []domain.Program{
			{Channel: "Ch1.src1", Title: "Show 1", Start: time.Now(), Stop: time.Now().Add(time.Hour)},
		},
	})

	plugin.Register(&fakeProvider{
		name: "src2",
		channels: []domain.Channel{
			{ID: "ch2", Name: "Ch2", Source: "src2", XmltvID: "Ch2.src2"},
		},
		programs: []domain.Program{
			{Channel: "Ch2.src2", Title: "Show 2", Start: time.Now(), Stop: time.Now().Add(time.Hour)},
		},
	})

	s := store.New("")
	g := grabber.New(s, 2, 1)

	results := g.GrabAll()
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestGrabConcurrency(t *testing.T) {
	plugin.Reset()

	fp := &fakeProvider{
		name: "concurrent-test",
		channels: []domain.Channel{
			{ID: "ch1", Name: "Ch1", Source: "concurrent-test", XmltvID: "Ch1.test"},
			{ID: "ch2", Name: "Ch2", Source: "concurrent-test", XmltvID: "Ch2.test"},
			{ID: "ch3", Name: "Ch3", Source: "concurrent-test", XmltvID: "Ch3.test"},
		},
		programs: []domain.Program{
			{Channel: "test", Title: "Show", Start: time.Now(), Stop: time.Now().Add(time.Hour)},
		},
	}
	plugin.Register(fp)

	s := store.New("")
	g := grabber.New(s, 3, 2)

	results, err := g.GrabSource("concurrent-test")
	if err != nil {
		t.Fatalf("grab failed: %v", err)
	}

	if len(results) != 6 {
		t.Fatalf("expected 6 results (3 channels * 2 days), got %d", len(results))
	}
}
