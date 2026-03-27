package api

import (
	"encoding/json"
	"net/http"

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

	json.NewEncoder(w).Encode(sources)
}
