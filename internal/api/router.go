package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/gavinmcnair/epg/internal/auth"
	"github.com/gavinmcnair/epg/internal/grabber"
	"github.com/gavinmcnair/epg/internal/source"
	"github.com/gavinmcnair/epg/internal/store"
)

type Server struct {
	store           *store.Store
	grabber         *grabber.Grabber
	externalSources *source.ExternalSourceManager
	users           *auth.UserStore
	router          chi.Router
}

func NewServer(s *store.Store, g *grabber.Grabber, ext *source.ExternalSourceManager, users *auth.UserStore) *Server {
	srv := &Server{
		store:           s,
		grabber:         g,
		externalSources: ext,
		users:           users,
		router:          chi.NewRouter(),
	}

	srv.router.Use(middleware.Logger)
	srv.router.Use(middleware.Recoverer)

	srv.router.Route("/api/v1", func(r chi.Router) {
		r.Use(users.Middleware)
		r.Use(middleware.SetHeader("Content-Type", "application/json"))

		r.Get("/sources", srv.listSources)
		r.Get("/channels", srv.listChannels)
		r.Get("/channels/{id}", srv.getChannel)
		r.Get("/programs", srv.listPrograms)
		r.Post("/grab", srv.triggerGrab)

		r.Get("/external-sources", srv.listExternalSources)
		r.Post("/external-sources", srv.addExternalSource)
		r.Delete("/external-sources/{name}", srv.removeExternalSource)
		r.Post("/external-sources/{name}/grab", srv.grabExternalSource)

		r.Get("/users", srv.listUsers)
		r.Post("/users", srv.createUser)
		r.Delete("/users/{id}", srv.deleteUser)
	})

	srv.router.Group(func(r chi.Router) {
		r.Use(users.Middleware)
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
