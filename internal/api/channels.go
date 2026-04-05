package api

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"

	"github.com/gavinmcnair/epg/internal/store"
)

func (s *Server) listChannels(w http.ResponseWriter, r *http.Request) {
	f := store.Filter{
		Source:  r.URL.Query().Get("source"),
		Genre:   r.URL.Query().Get("genre"),
		Quality: r.URL.Query().Get("quality"),
		Region:  r.URL.Query().Get("region"),
		Lang:    r.URL.Query().Get("lang"),
		Search:  r.URL.Query().Get("search"),
	}

	channels := s.store.GetChannels(f)
	json.NewEncoder(w).Encode(channels)
}

func (s *Server) getChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	ch, ok := s.store.GetChannel(id)
	if !ok {
		http.Error(w, `{"error":"channel not found"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(ch)
}

func (s *Server) listRegions(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	f := store.Filter{Source: source}
	channels := s.store.GetChannels(f)

	regionSet := make(map[string]bool)
	for _, ch := range channels {
		if ch.Region != "" {
			regionSet[ch.Region] = true
		}
	}

	regions := make([]string, 0, len(regionSet))
	for r := range regionSet {
		regions = append(regions, r)
	}
	sort.Strings(regions)

	json.NewEncoder(w).Encode(regions)
}
