package api

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
)

type sleepRequest struct {
	Reason string `json:"reason"`
}

func (s *Server) listSleeping(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.sleeping.List())
}

func (s *Server) sleepSource(w http.ResponseWriter, r *http.Request) {
	name, _ := url.PathUnescape(chi.URLParam(r, "name"))

	var req sleepRequest
	json.NewDecoder(r.Body).Decode(&req)

	if req.Reason == "" {
		req.Reason = "manually slept"
	}

	s.sleeping.Sleep(name, req.Reason)
	s.sleeping.Save()

	json.NewEncoder(w).Encode(map[string]string{"status": "sleeping", "name": name, "reason": req.Reason})
}

func (s *Server) wakeSource(w http.ResponseWriter, r *http.Request) {
	name, _ := url.PathUnescape(chi.URLParam(r, "name"))

	s.sleeping.Wake(name)
	s.sleeping.Save()

	json.NewEncoder(w).Encode(map[string]string{"status": "awake", "name": name})
}
