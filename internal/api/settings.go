package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	all := s.settings.All()
	safe := make(map[string]string)
	for k, v := range all {
		if k == "tmdb_api_key" && len(v) > 8 {
			safe[k] = v[:4] + "..." + v[len(v)-4:]
		} else {
			safe[k] = v
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(safe)
}

func (s *Server) putSettings(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	allowed := map[string]bool{"tmdb_api_key": true}
	for k, v := range body {
		if !allowed[k] {
			continue
		}
		if err := s.settings.Set(k, v); err != nil {
			jsonError(w, "failed to save setting", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
