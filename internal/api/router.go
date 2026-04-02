package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/gavinmcnair/epg/internal/auth"
	"github.com/gavinmcnair/epg/internal/bandwidth"
	"github.com/gavinmcnair/epg/internal/grabber"
	"github.com/gavinmcnair/epg/internal/logocache"
	"github.com/gavinmcnair/epg/internal/source"
	"github.com/gavinmcnair/epg/internal/store"
	"github.com/gavinmcnair/epg/internal/ui"
)

type Server struct {
	store           *store.Store
	grabber         *grabber.Grabber
	externalSources *source.ExternalSourceManager
	users           *auth.UserStore
	disabled        *store.DisabledStore
	logoCache       *logocache.Cache
	bandwidth       *bandwidth.Tracker
	router          chi.Router
}

func NewServer(s *store.Store, g *grabber.Grabber, ext *source.ExternalSourceManager, users *auth.UserStore, disabled *store.DisabledStore, logos *logocache.Cache, bw *bandwidth.Tracker) *Server {
	srv := &Server{
		store:           s,
		grabber:         g,
		externalSources: ext,
		users:           users,
		disabled:        disabled,
		logoCache:       logos,
		bandwidth:       bw,
		router:          chi.NewRouter(),
	}

	srv.router.Use(middleware.Logger)
	srv.router.Use(middleware.Recoverer)

	srv.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	srv.router.Get("/api/openapi.yaml", serveOpenAPI)

	srv.router.Get("/login", srv.loginPage)
	srv.router.Post("/login", srv.login)
	srv.router.Post("/logout", srv.logout)

	// Read-only API: token (Bearer or ?token=) or admin session cookie
	srv.router.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.SetHeader("Content-Type", "application/json"))

		r.Group(func(r chi.Router) {
			r.Use(users.Middleware)
			r.Get("/sources", srv.listSources)
			r.Get("/channels", srv.listChannels)
			r.Get("/channels/{id}", srv.getChannel)
			r.Get("/regions", srv.listRegions)
			r.Get("/programs", srv.listPrograms)
			r.Get("/stats", srv.getStats)
			r.Get("/grab-stats", srv.getGrabStats)
			r.Get("/grab-progress", srv.getGrabProgress)
			r.Get("/bandwidth", srv.getBandwidth)
		})

		// Admin-only: session cookie required
		r.Group(func(r chi.Router) {
			r.Use(users.AdminAPIMiddleware)
			r.Post("/grab", srv.triggerGrab)
			r.Post("/grab-all", srv.triggerGrabAll)

			r.Get("/external-sources", srv.listExternalSources)
			r.Post("/external-sources", srv.addExternalSource)
			r.Delete("/external-sources/{name}", srv.removeExternalSource)
			r.Post("/external-sources/{name}/grab", srv.grabExternalSource)

			r.Get("/disabled", srv.listDisabled)
			r.Post("/disabled/{name}", srv.disableSource)
			r.Delete("/disabled/{name}", srv.enableSource)

			r.Get("/users", srv.listUsers)
			r.Post("/users", srv.createUser)
			r.Put("/users/{id}/password", srv.changePassword)
			r.Delete("/users/{id}", srv.deleteUser)
		})
	})

	// Logos: public (IPTV players don't send auth)
	srv.router.Get("/logos/{id}", srv.serveLogo)

	// XMLTV: token or admin session cookie
	srv.router.Group(func(r chi.Router) {
		r.Use(users.Middleware)
		r.Use(middleware.SetHeader("Content-Type", "application/xml"))
		r.Get("/epg.xml", srv.getXMLTV)
		r.Get("/epg/*", srv.getSourceXMLTV)
	})

	// Admin UI: session cookie required
	srv.router.Group(func(r chi.Router) {
		r.Use(users.AdminMiddleware)
		r.Handle("/*", ui.Handler())
	})

	return srv
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) Handler() http.Handler {
	return s.router
}
