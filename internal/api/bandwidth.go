package api

import (
	"encoding/json"
	"net/http"

	"github.com/gavinmcnair/epg/internal/bandwidth"
)

func (s *Server) getBandwidth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]any{
		"rate_bytes_per_sec": bandwidth.Global.Rate(),
		"total_bytes":        bandwidth.Global.Total(),
	})
}
