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
	plugins := plugin.List()
	externalSources := s.externalSources.List()
	resp := statsResponse{
		Sources:         len(plugins) + len(externalSources),
		Channels:        s.store.ChannelCount(),
		Programs:        s.store.ProgramCount(),
		Plugins:         len(plugins),
		ExternalSources: len(externalSources),
		GrabStats:       s.store.GetGrabStats(),
	}
	json.NewEncoder(w).Encode(resp)
}
