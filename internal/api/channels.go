package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gavinmcnair/epg/internal/store"
)

func (s *Server) listChannels(w http.ResponseWriter, r *http.Request) {
	f := store.Filter{
		Source:  r.URL.Query().Get("source"),
		Genre:   r.URL.Query().Get("genre"),
		Quality: r.URL.Query().Get("quality"),
		Lang:    r.URL.Query().Get("lang"),
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
