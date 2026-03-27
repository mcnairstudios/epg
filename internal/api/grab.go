package api

import (
	"encoding/json"
	"net/http"
	"time"
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
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Source == "" {
		http.Error(w, `{"error":"source is required"}`, http.StatusBadRequest)
		return
	}

	start := time.Now()
	results, err := s.grabber.GrabSource(req.Source)
	duration := time.Since(start)

	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	resp := grabResponse{Source: req.Source}
	channelSet := make(map[string]bool)
	for _, result := range results {
		resp.Results += result.Count
		if result.Err != nil {
			resp.Errors++
		}
		if result.Count > 0 {
			channelSet[result.Channel.ID] = true
		}
	}

	s.store.RecordGrabStats(req.Source, len(channelSet), resp.Results, resp.Errors, duration)

	if err := s.store.Save(); err != nil {
		http.Error(w, `{"error":"failed to persist data"}`, http.StatusInternalServerError)
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
	results := s.grabber.GrabAll()

	sourceResults := make(map[string]*grabResponse)
	sourceChannelSets := make(map[string]map[string]bool)
	sourceTimes := make(map[string]time.Time)

	for _, result := range results {
		sr, ok := sourceResults[result.Source]
		if !ok {
			sr = &grabResponse{Source: result.Source}
			sourceResults[result.Source] = sr
			sourceChannelSets[result.Source] = make(map[string]bool)
			sourceTimes[result.Source] = time.Now()
		}
		sr.Results += result.Count
		if result.Err != nil {
			sr.Errors++
		}
		if result.Count > 0 {
			sourceChannelSets[result.Source][result.Channel.ID] = true
		}
	}

	resp := grabAllResponse{}
	for name, sr := range sourceResults {
		resp.Sources = append(resp.Sources, *sr)
		resp.Total += sr.Results
		resp.Errors += sr.Errors
		s.store.RecordGrabStats(name, len(sourceChannelSets[name]), sr.Results, sr.Errors, time.Since(sourceTimes[name]))
	}

	s.store.Save()

	json.NewEncoder(w).Encode(resp)
}

func (s *Server) getGrabStats(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.store.GetGrabStats())
}
