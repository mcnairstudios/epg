package api

import (
	"encoding/json"
	"net/http"

	"github.com/gavinmcnair/epg/internal/plugin"
	"github.com/gavinmcnair/epg/internal/store"
)

type statsResponse struct {
	Sources         int              `json:"sources"`
	Channels        int              `json:"channels"`
	Programs        int              `json:"programs"`
	Plugins         int              `json:"plugins"`
	ExternalSources int              `json:"external_sources"`
	GrabStats       []store.GrabStats `json:"grab_stats,omitempty"`
}

func (s *Server) getStats(w http.ResponseWriter, r *http.Request) {
	resp := statsResponse{
		Sources:         len(plugin.List()) + len(s.externalSources.List()),
		Channels:        s.store.ChannelCount(),
		Programs:        s.store.ProgramCount(),
		Plugins:         len(plugin.List()),
		ExternalSources: len(s.externalSources.List()),
		GrabStats:       s.grabStats.All(),
	}
	json.NewEncoder(w).Encode(resp)
}
