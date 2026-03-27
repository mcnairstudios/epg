package api

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/plugin"
)

func (s *Server) listSources(w http.ResponseWriter, r *http.Request) {
	providers := plugin.List()

	sources := make([]domain.Source, 0, len(providers))
	for _, p := range providers {
		sources = append(sources, domain.Source{
			Name:        p.Name(),
			Description: p.Description(),
			Type:        "plugin",
		})
	}

	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Name < sources[j].Name
	})

	json.NewEncoder(w).Encode(sources)
}
