package api

import (
	"net/http"

	"github.com/gavinmcnair/epg/internal/ui"
)

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("epg_session"); err == nil {
		if _, ok := s.users.ValidateSession(cookie.Value); ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}
	ui.ServeLogin(w, r)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, ok := s.users.ValidatePassword(username, password)
	if !ok {
		http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
		return
	}

	token := s.users.CreateSession(user.ID)
	http.SetCookie(w, &http.Cookie{
		Name:     "epg_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("epg_session"); err == nil {
		s.users.DeleteSession(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "epg_session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
