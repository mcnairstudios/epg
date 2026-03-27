package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/gavinmcnair/epg/internal/store"
	"github.com/gavinmcnair/epg/internal/xmltv"
)

func (s *Server) getXMLTV(w http.ResponseWriter, r *http.Request) {
	f := store.Filter{
		Source:  r.URL.Query().Get("source"),
		Channel: r.URL.Query().Get("channel"),
		Genre:   r.URL.Query().Get("genre"),
		Quality: r.URL.Query().Get("quality"),
	}

	s.writeXMLTV(w, f)
}

func (s *Server) getSourceXMLTV(w http.ResponseWriter, r *http.Request) {
	source := chi.URLParam(r, "source")
	f := store.Filter{
		Source:  source,
		Channel: r.URL.Query().Get("channel"),
		Genre:   r.URL.Query().Get("genre"),
		Quality: r.URL.Query().Get("quality"),
	}

	s.writeXMLTV(w, f)
}

func (s *Server) writeXMLTV(w http.ResponseWriter, f store.Filter) {
	channels := s.store.GetChannels(f)

	programFilter := store.Filter{
		Source:  f.Source,
		Channel: f.Channel,
		Genre:   f.Genre,
	}
	if f.Date != nil {
		programFilter.Date = f.Date
	} else {
		now := time.Now().UTC().Truncate(24 * time.Hour)
		programFilter.Date = &now
	}

	programs := s.store.GetPrograms(programFilter)

	data, err := xmltv.Generate(channels, programs)
	if err != nil {
		http.Error(w, "failed to generate XMLTV", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write(data)
}
