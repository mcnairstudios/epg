package auth_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gavinmcnair/epg/internal/auth"
)

func TestCreateUser(t *testing.T) {
	s := auth.NewUserStore("")

	user, err := s.Create("testuser", "admin")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if user.Username != "testuser" {
		t.Fatalf("expected username 'testuser', got %q", user.Username)
	}
	if user.Role != "admin" {
		t.Fatalf("expected role 'admin', got %q", user.Role)
	}
	if user.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if user.ID == "" {
		t.Fatal("expected non-empty ID")
	}
}

func TestCreateDuplicateUsername(t *testing.T) {
	s := auth.NewUserStore("")

	s.Create("testuser", "admin")
	_, err := s.Create("testuser", "viewer")
	if err == nil {
		t.Fatal("expected error on duplicate username")
	}
}

func TestDeleteUser(t *testing.T) {
	s := auth.NewUserStore("")

	user, _ := s.Create("testuser", "admin")
	if err := s.Delete(user.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	users := s.List()
	if len(users) != 0 {
		t.Fatalf("expected 0 users after delete, got %d", len(users))
	}
}

func TestDeleteNonexistent(t *testing.T) {
	s := auth.NewUserStore("")

	if err := s.Delete("nonexistent"); err == nil {
		t.Fatal("expected error on nonexistent delete")
	}
}

func TestValidateToken(t *testing.T) {
	s := auth.NewUserStore("")

	user, _ := s.Create("testuser", "admin")

	found, ok := s.ValidateToken(user.Token)
	if !ok {
		t.Fatal("expected token to be valid")
	}
	if found.Username != "testuser" {
		t.Fatalf("expected username 'testuser', got %q", found.Username)
	}

	_, ok = s.ValidateToken("invalid-token")
	if ok {
		t.Fatal("expected invalid token to fail")
	}
}

func TestMiddlewareNoUsers(t *testing.T) {
	s := auth.NewUserStore("")

	handler := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with no users, got %d", w.Code)
	}
}

func TestMiddlewareWithUsers(t *testing.T) {
	s := auth.NewUserStore("")

	user, _ := s.Create("testuser", "admin")

	handler := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+user.Token)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/?token="+user.Token, nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with query param token, got %d", w.Code)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.gob")

	s := auth.NewUserStore(path)
	s.Create("user1", "admin")
	s.Create("user2", "viewer")

	if err := s.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	s2 := auth.NewUserStore(path)
	if err := s2.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	users := s2.List()
	if len(users) != 2 {
		t.Fatalf("expected 2 users after load, got %d", len(users))
	}
}
