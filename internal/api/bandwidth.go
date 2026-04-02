package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) getBandwidth(w http.ResponseWriter, r *http.Request) {
	if s.bandwidth == nil {
		json.NewEncoder(w).Encode(map[string]any{
			"rate_bytes_per_sec": 0,
			"total_bytes":        0,
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"rate_bytes_per_sec": s.bandwidth.Rate(),
		"total_bytes":        s.bandwidth.Total(),
	})
}
