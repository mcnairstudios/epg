package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/gavinmcnair/epg/internal/grabber"
	"github.com/gavinmcnair/epg/internal/store"
)

type Server struct {
	store   *store.Store
	grabber *grabber.Grabber
	router  chi.Router
}

func NewServer(s *store.Store, g *grabber.Grabber) *Server {
	srv := &Server{
		store:   s,
		grabber: g,
		router:  chi.NewRouter(),
	}

	srv.router.Use(middleware.Logger)
	srv.router.Use(middleware.Recoverer)
	srv.router.Use(middleware.SetHeader("Content-Type", "application/json"))

	srv.router.Route("/api/v1", func(r chi.Router) {
		r.Get("/sources", srv.listSources)
		r.Get("/channels", srv.listChannels)
		r.Get("/channels/{id}", srv.getChannel)
		r.Get("/programs", srv.listPrograms)
		r.Post("/grab", srv.triggerGrab)
	})

	srv.router.Group(func(r chi.Router) {
		r.Use(middleware.SetHeader("Content-Type", "application/xml"))
		r.Get("/epg.xml", srv.getXMLTV)
		r.Get("/epg/{source}.xml", srv.getSourceXMLTV)
	})

	return srv
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) Handler() http.Handler {
	return s.router
}
