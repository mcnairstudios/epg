package api

import (
	"encoding/csv"
	"net/http"
	"sort"

	"github.com/gavinmcnair/epg/internal/store"
)

func (s *Server) getMapping(w http.ResponseWriter, r *http.Request) {
	f := store.Filter{
		Source:  r.URL.Query().Get("source"),
		Genre:   r.URL.Query().Get("genre"),
		Quality: r.URL.Query().Get("quality"),
		Region:  r.URL.Query().Get("region"),
		Lang:    r.URL.Query().Get("lang"),
		Search:  r.URL.Query().Get("search"),
	}

	channels := s.store.GetChannels(f)
	sort.Slice(channels, func(i, j int) bool {
		if channels[i].Name != channels[j].Name {
			return channels[i].Name < channels[j].Name
		}
		return channels[i].Source < channels[j].Source
	})

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="epg-mapping.csv"`)

	cw := csv.NewWriter(w)
	cw.Write([]string{"name", "xmltv_id", "source", "logo"})
	for _, ch := range channels {
		if ch.XmltvID == "" {
			continue
		}
		cw.Write([]string{ch.Name, ch.XmltvID, ch.Source, ch.Logo})
	}
	cw.Flush()
}
