package api

import (
	"compress/gzip"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/store"
	"github.com/gavinmcnair/epg/internal/xmltv"
)

func logoKey(xmltvID string) string {
	if i := strings.Index(xmltvID, "@"); i != -1 {
		return xmltvID[:i]
	}
	return xmltvID
}

func (s *Server) getXMLTV(w http.ResponseWriter, r *http.Request) {
	f := store.Filter{
		Source:  r.URL.Query().Get("source"),
		Channel: r.URL.Query().Get("channel"),
		Genre:   r.URL.Query().Get("genre"),
		Quality: r.URL.Query().Get("quality"),
	}

	s.writeXMLTV(w, r, f)
}

func (s *Server) getSourceXMLTV(w http.ResponseWriter, r *http.Request) {
	source := chi.URLParam(r, "*")
	source = strings.TrimSuffix(source, ".xml")
	source = strings.TrimPrefix(source, "/")
	f := store.Filter{
		Source:  source,
		Channel: r.URL.Query().Get("channel"),
		Genre:   r.URL.Query().Get("genre"),
		Quality: r.URL.Query().Get("quality"),
	}

	s.writeXMLTV(w, r, f)
}

func (s *Server) writeXMLTV(w http.ResponseWriter, r *http.Request, f store.Filter) {
	disabled := s.disabled.List()

	allChannels := s.store.GetChannels(f)
	channels := make([]domain.Channel, 0, len(allChannels))
	for _, ch := range allChannels {
		if disabled[ch.Source] {
			continue
		}
		if ch.Logo != "" {
			scheme := "https"
			if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") == "" {
				scheme = "http"
			} else if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
				scheme = proto
			}
			ch.Logo = scheme + "://" + r.Host + "/logos/" + logoKey(ch.XmltvID)
		}
		channels = append(channels, ch)
	}

	programFilter := store.Filter{
		Source:  f.Source,
		Channel: f.Channel,
		Genre:   f.Genre,
		Date:    f.Date,
	}

	allPrograms := s.store.GetPrograms(programFilter)
	programs := allPrograms[:0]
	for _, p := range allPrograms {
		if !disabled[p.Source] {
			programs = append(programs, p)
		}
	}

	data, err := xmltv.Generate(channels, programs)
	if err != nil {
		http.Error(w, "failed to generate XMLTV", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")

	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gz.Write(data)
		return
	}

	w.Write(data)
}
