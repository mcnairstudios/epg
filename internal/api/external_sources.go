package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gavinmcnair/epg/internal/domain"
)

type addSourceRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (s *Server) listExternalSources(w http.ResponseWriter, r *http.Request) {
	sources := s.externalSources.List()
	json.NewEncoder(w).Encode(sources)
}

func (s *Server) addExternalSource(w http.ResponseWriter, r *http.Request) {
	var req addSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.URL == "" {
		http.Error(w, `{"error":"name and url are required"}`, http.StatusBadRequest)
		return
	}

	if err := s.externalSources.Add(req.Name, req.URL); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusConflict)
		return
	}

	if err := s.externalSources.Save(); err != nil {
		http.Error(w, `{"error":"failed to persist"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "added", "name": req.Name})
}

func (s *Server) removeExternalSource(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := s.externalSources.Remove(name); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}

	if err := s.externalSources.Save(); err != nil {
		http.Error(w, `{"error":"failed to persist"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "removed", "name": name})
}

func (s *Server) grabExternalSource(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	channels, programs, err := s.externalSources.Fetch(name)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	s.store.SetChannels(channels)
	for _, ch := range channels {
		var chPrograms []domain.Program
		for _, p := range programs {
			if p.Channel == ch.XmltvID {
				chPrograms = append(chPrograms, p)
			}
		}
		if len(chPrograms) > 0 {
			s.store.SetPrograms(ch.ID, chPrograms)
		}
	}

	if err := s.store.Save(); err != nil {
		http.Error(w, `{"error":"failed to persist data"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"source":   name,
		"channels": len(channels),
		"programs": len(programs),
	})
}
