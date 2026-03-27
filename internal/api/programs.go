package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gavinmcnair/epg/internal/store"
)

func (s *Server) listPrograms(w http.ResponseWriter, r *http.Request) {
	f := store.Filter{
		Source:  r.URL.Query().Get("source"),
		Channel: r.URL.Query().Get("channel"),
		Genre:   r.URL.Query().Get("genre"),
	}

	if dateStr := r.URL.Query().Get("date"); dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			f.Date = &t
		}
	}

	programs := s.store.GetPrograms(f)
	json.NewEncoder(w).Encode(programs)
}
