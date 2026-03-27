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

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	users := s.users.List()
	json.NewEncoder(w).Encode(users)
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Username == "" {
		http.Error(w, `{"error":"username is required"}`, http.StatusBadRequest)
		return
	}

	if req.Role == "" {
		req.Role = "viewer"
	}

	user, err := s.users.Create(req.Username, req.Role)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusConflict)
		return
	}

	if err := s.users.Save(); err != nil {
		http.Error(w, `{"error":"failed to persist"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.users.Delete(id); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}

	if err := s.users.Save(); err != nil {
		http.Error(w, `{"error":"failed to persist"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
