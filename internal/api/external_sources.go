package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/gavinmcnair/epg/internal/domain"
)

type addSourceRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type externalSourceResponse struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Type     string `json:"type"`
	Disabled bool   `json:"disabled"`
}

func (s *Server) listExternalSources(w http.ResponseWriter, r *http.Request) {
	sources := s.externalSources.List()
	result := make([]externalSourceResponse, 0, len(sources))
	for _, src := range sources {
		result = append(result, externalSourceResponse{
			Name:     src.Name,
			URL:      src.URL,
			Type:     src.Type,
			Disabled: s.disabled.IsDisabled(src.Name),
		})
	}
	json.NewEncoder(w).Encode(result)
}

func (s *Server) addExternalSource(w http.ResponseWriter, r *http.Request) {
	var req addSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.URL == "" {
		jsonError(w, "name and url are required", http.StatusBadRequest)
		return
	}

	if err := s.externalSources.Add(req.Name, req.URL); err != nil {
		jsonError(w, err.Error(), http.StatusConflict)
		return
	}

	if err := s.externalSources.Save(); err != nil {
		jsonError(w, "failed to persist", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "added", "name": req.Name})
}

func (s *Server) removeExternalSource(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := s.externalSources.Remove(name); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := s.externalSources.Save(); err != nil {
		jsonError(w, "failed to persist", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "removed", "name": name})
}

func (s *Server) grabExternalSource(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	start := time.Now()
	channels, programs, err := s.externalSources.Fetch(r.Context(), name)
	duration := time.Since(start)

	if err != nil {
		s.log.Error().Err(err).Str("source", name).Msg("external grab failed")
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.store.SetChannels(channels)

	prog := s.grabber.Progress()
	prog.Start(name, len(channels))

	channelsWithData := 0
	for _, ch := range channels {
		s.store.ClearPrograms(ch.ID)
		var chPrograms []domain.Program
		for _, p := range programs {
			if p.Channel == ch.XmltvID {
				chPrograms = append(chPrograms, p)
			}
		}
		hasData := len(chPrograms) > 0
		if hasData {
			s.store.SetPrograms(ch.ID, chPrograms)
			channelsWithData++
		}
		prog.Record(name, !hasData)
	}

	prog.Finish(name)
	s.store.RecordGrabStats(name, len(channels), channelsWithData, len(programs), 0, duration)

	if err := s.store.SaveSource(name); err != nil {
		jsonError(w, "failed to persist data", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"source":   name,
		"channels": len(channels),
		"programs": len(programs),
	})
}
