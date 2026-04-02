package api

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
)

func (s *Server) listDisabled(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.disabled.List())
}

func (s *Server) disableSource(w http.ResponseWriter, r *http.Request) {
	name, _ := url.PathUnescape(chi.URLParam(r, "name"))
	s.disabled.Disable(name)
	if err := s.disabled.Save(); err != nil {
		http.Error(w, `{"error":"failed to persist"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "disabled", "name": name})
}

func (s *Server) enableSource(w http.ResponseWriter, r *http.Request) {
	name, _ := url.PathUnescape(chi.URLParam(r, "name"))
	s.disabled.Enable(name)
	if err := s.disabled.Save(); err != nil {
		http.Error(w, `{"error":"failed to persist"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "enabled", "name": name})
}
