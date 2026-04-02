package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gavinmcnair/epg/internal/grabber"
)

type grabRequest struct {
	Source string `json:"source"`
}

type grabResponse struct {
	Source  string `json:"source"`
	Results int    `json:"results"`
	Errors  int    `json:"errors"`
}

func (s *Server) triggerGrab(w http.ResponseWriter, r *http.Request) {
	var req grabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Source == "" {
		jsonError(w, "source is required", http.StatusBadRequest)
		return
	}

	start := time.Now()
	results, err := s.grabber.GrabSource(r.Context(), req.Source)
	duration := time.Since(start)

	if err != nil {
		jsonError(w, "grab failed", http.StatusBadRequest)
		return
	}

	summary := grabber.Summarize(results)
	resp := grabResponse{Source: req.Source, Results: summary.Programs, Errors: summary.Errors}

	s.store.RecordGrabStats(req.Source, summary.TotalChannels, summary.ChannelsWithData, resp.Results, resp.Errors, duration)

	if err := s.store.SaveSource(req.Source); err != nil {
		jsonError(w, "failed to persist data", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

type grabAllResponse struct {
	Sources []grabResponse `json:"sources"`
	Total   int            `json:"total"`
	Errors  int            `json:"errors"`
}

func (s *Server) triggerGrabAll(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	results := s.grabber.GrabAll(r.Context(), s.disabled.List())
	duration := time.Since(start)

	bySource := make(map[string][]grabber.Result)
	for _, result := range results {
		bySource[result.Source] = append(bySource[result.Source], result)
	}

	resp := grabAllResponse{}
	for name, sourceResults := range bySource {
		summary := grabber.Summarize(sourceResults)
		sr := grabResponse{Source: name, Results: summary.Programs, Errors: summary.Errors}
		resp.Sources = append(resp.Sources, sr)
		resp.Total += sr.Results
		resp.Errors += sr.Errors
		s.store.RecordGrabStats(name, summary.TotalChannels, summary.ChannelsWithData, sr.Results, sr.Errors, duration)
	}

	if err := s.store.Save(); err != nil {
		s.log.Error().Err(err).Msg("grab-all: failed to persist")
		jsonError(w, "failed to persist grab results", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) getGrabStats(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.store.GetGrabStats())
}

func (s *Server) getGrabProgress(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.grabber.Progress().All())
}
