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

func TestSetAndValidatePassword(t *testing.T) {
	s := auth.NewUserStore("")

	user, _ := s.Create("admin", "admin")

	if err := s.SetPassword(user.ID, "secret123"); err != nil {
		t.Fatalf("SetPassword failed: %v", err)
	}

	found, ok := s.ValidatePassword("admin", "secret123")
	if !ok {
		t.Fatal("expected password to be valid")
	}
	if found.Username != "admin" {
		t.Fatalf("expected username 'admin', got %q", found.Username)
	}

	_, ok = s.ValidatePassword("admin", "wrongpassword")
	if ok {
		t.Fatal("expected wrong password to fail")
	}
}

func TestValidatePasswordNoPasswordSet(t *testing.T) {
	s := auth.NewUserStore("")

	s.Create("admin", "admin")

	_, ok := s.ValidatePassword("admin", "anything")
	if ok {
		t.Fatal("expected validation to fail when no password set")
	}
}

func TestSetPasswordNonexistentUser(t *testing.T) {
	s := auth.NewUserStore("")

	if err := s.SetPassword("nonexistent", "password"); err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestHasAdminWithPassword(t *testing.T) {
	s := auth.NewUserStore("")

	if s.HasAdminWithPassword() {
		t.Fatal("expected false with no users")
	}

	user, _ := s.Create("admin", "admin")
	if s.HasAdminWithPassword() {
		t.Fatal("expected false with admin but no password")
	}

	s.SetPassword(user.ID, "secret")
	if !s.HasAdminWithPassword() {
		t.Fatal("expected true after setting password")
	}
}

func TestEnsureAdminCreatesNewAdmin(t *testing.T) {
	s := auth.NewUserStore("")

	if err := s.EnsureAdmin("admin", "password"); err != nil {
		t.Fatalf("EnsureAdmin failed: %v", err)
	}

	_, ok := s.ValidatePassword("admin", "password")
	if !ok {
		t.Fatal("expected to validate password after EnsureAdmin")
	}

	if !s.HasAdminWithPassword() {
		t.Fatal("expected HasAdminWithPassword to be true")
	}
}

func TestEnsureAdminSetsPasswordOnExistingAdmin(t *testing.T) {
	s := auth.NewUserStore("")

	s.Create("existing-admin", "admin")

	if err := s.EnsureAdmin("admin", "newpassword"); err != nil {
		t.Fatalf("EnsureAdmin failed: %v", err)
	}

	_, ok := s.ValidatePassword("existing-admin", "newpassword")
	if !ok {
		t.Fatal("expected password set on existing passwordless admin")
	}

	users := s.List()
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d (should not create new admin when one exists)", len(users))
	}
}

func TestSessions(t *testing.T) {
	s := auth.NewUserStore("")

	user, _ := s.Create("admin", "admin")

	token := s.CreateSession(user.ID)
	if token == "" {
		t.Fatal("expected non-empty session token")
	}

	found, ok := s.ValidateSession(token)
	if !ok {
		t.Fatal("expected session to be valid")
	}
	if found.Username != "admin" {
		t.Fatalf("expected username 'admin', got %q", found.Username)
	}

	s.DeleteSession(token)

	_, ok = s.ValidateSession(token)
	if ok {
		t.Fatal("expected session to be invalid after deletion")
	}
}

func TestSessionInvalidToken(t *testing.T) {
	s := auth.NewUserStore("")

	_, ok := s.ValidateSession("nonexistent-token")
	if ok {
		t.Fatal("expected invalid session token to fail")
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

func TestMiddlewareAcceptsSession(t *testing.T) {
	s := auth.NewUserStore("")

	user, _ := s.Create("admin", "admin")
	sessionToken := s.CreateSession(user.ID)

	handler := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "epg_session", Value: sessionToken})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid session cookie, got %d", w.Code)
	}
}

func TestAdminAPIMiddleware(t *testing.T) {
	s := auth.NewUserStore("")

	user, _ := s.Create("admin", "admin")

	handler := s.AdminAPIMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/grab", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/grab", nil)
	req.Header.Set("Authorization", "Bearer "+user.Token)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with token (not session), got %d", w.Code)
	}

	sessionToken := s.CreateSession(user.ID)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/grab", nil)
	req.AddCookie(&http.Cookie{Name: "epg_session", Value: sessionToken})
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid session, got %d", w.Code)
	}
}

func TestAdminMiddlewareRedirects(t *testing.T) {
	s := auth.NewUserStore("")

	user, _ := s.Create("admin", "admin")

	handler := s.AdminMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect without session, got %d", w.Code)
	}
	if w.Header().Get("Location") != "/login" {
		t.Fatalf("expected redirect to /login, got %q", w.Header().Get("Location"))
	}

	sessionToken := s.CreateSession(user.ID)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "epg_session", Value: sessionToken})
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid session, got %d", w.Code)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.gob")

	s := auth.NewUserStore(path)
	user1, _ := s.Create("user1", "admin")
	s.SetPassword(user1.ID, "password1")
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

	_, ok := s2.ValidatePassword("user1", "password1")
	if !ok {
		t.Fatal("expected password to survive save/load")
	}
}
