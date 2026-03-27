package seed

import (
	_ "embed"
	"encoding/json"

	"github.com/gavinmcnair/epg/internal/source"
)

//go:embed externals.json
var defaultExternals []byte

type externalEntry struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func SeedExternalSources(mgr *source.ExternalSourceManager) int {
	var entries []externalEntry
	if err := json.Unmarshal(defaultExternals, &entries); err != nil {
		return 0
	}

	added := 0
	for _, e := range entries {
		if _, exists := mgr.Get(e.Name); exists {
			continue
		}
		if err := mgr.Add(e.Name, e.URL); err == nil {
			added++
		}
	}

	if added > 0 {
		mgr.Save()
	}

	return added
}
