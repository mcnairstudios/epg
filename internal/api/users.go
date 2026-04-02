package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type createUserRequest struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

type changePasswordRequest struct {
	Password string `json:"password"`
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	users := s.users.List()
	json.NewEncoder(w).Encode(users)
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		jsonError(w, "username is required", http.StatusBadRequest)
		return
	}

	if req.Role == "" {
		req.Role = "viewer"
	}

	user, err := s.users.Create(req.Username, req.Role)
	if err != nil {
		jsonError(w, err.Error(), http.StatusConflict)
		return
	}

	if err := s.users.Save(); err != nil {
		jsonError(w, "failed to persist", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Password == "" {
		jsonError(w, "password is required", http.StatusBadRequest)
		return
	}

	if err := s.users.SetPassword(id, req.Password); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := s.users.Save(); err != nil {
		jsonError(w, "failed to persist", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "password changed"})
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.users.Delete(id); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := s.users.Save(); err != nil {
		jsonError(w, "failed to persist", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
