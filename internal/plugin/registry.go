package plugin

import (
	"fmt"
	"sort"
	"sync"
)

var (
	mu        sync.RWMutex
	providers = make(map[string]Provider)
)

func Register(p Provider) {
	mu.Lock()
	defer mu.Unlock()

	name := p.Name()
	if _, exists := providers[name]; exists {
		panic(fmt.Sprintf("plugin already registered: %s", name))
	}
	providers[name] = p
}

func Get(name string) (Provider, bool) {
	mu.RLock()
	defer mu.RUnlock()

	p, ok := providers[name]
	return p, ok
}

func List() []Provider {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]Provider, 0, len(providers))
	for _, name := range names {
		result = append(result, providers[name])
	}
	return result
}

func Names() []string {
	mu.RLock()
	defer mu.RUnlock()

	result := make([]string, 0, len(providers))
	for name := range providers {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func Reset() {
	mu.Lock()
	defer mu.Unlock()

	providers = make(map[string]Provider)
}
