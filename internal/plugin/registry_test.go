package plugin_test

import (
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string        { return m.name }
func (m *mockProvider) Description() string { return "mock provider" }
func (m *mockProvider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{{ID: "ch1", Name: "Test Channel"}}, nil
}
func (m *mockProvider) Fetch(channel domain.Channel, date time.Time) ([]domain.Program, error) {
	return nil, nil
}

func TestRegisterAndGet(t *testing.T) {
	plugin.Reset()

	p := &mockProvider{name: "test-provider"}
	plugin.Register(p)

	got, ok := plugin.Get("test-provider")
	if !ok {
		t.Fatal("expected provider to be found")
	}
	if got.Name() != "test-provider" {
		t.Fatalf("expected name test-provider, got %s", got.Name())
	}
}

func TestGetMissing(t *testing.T) {
	plugin.Reset()

	_, ok := plugin.Get("nonexistent")
	if ok {
		t.Fatal("expected provider not to be found")
	}
}

func TestList(t *testing.T) {
	plugin.Reset()

	plugin.Register(&mockProvider{name: "a"})
	plugin.Register(&mockProvider{name: "b"})

	providers := plugin.List()
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}
}

func TestNames(t *testing.T) {
	plugin.Reset()

	plugin.Register(&mockProvider{name: "x"})
	plugin.Register(&mockProvider{name: "y"})

	names := plugin.Names()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
}

func TestDuplicateRegisterPanics(t *testing.T) {
	plugin.Reset()

	plugin.Register(&mockProvider{name: "dup"})

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()

	plugin.Register(&mockProvider{name: "dup"})
}
