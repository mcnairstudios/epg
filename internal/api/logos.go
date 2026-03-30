package api

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
)

func (s *Server) serveLogo(w http.ResponseWriter, r *http.Request) {
	id, _ := url.PathUnescape(chi.URLParam(r, "id"))
	s.logoCache.Serve(w, r, id)
}
