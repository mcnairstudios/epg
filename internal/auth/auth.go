package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var dummyHash = func() []byte {
	h, _ := bcrypt.GenerateFromPassword([]byte("dummy-constant-placeholder"), bcrypt.DefaultCost)
	return h
}()

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Token        string    `json:"token"`
	Role         string    `json:"role"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Session struct {
	UserID  string
	Expires time.Time
}

type UserStore struct {
	mu       sync.RWMutex
	users    map[string]User
	sessions map[string]Session
	path     string
}

type userSnapshot struct {
	Users map[string]User
}

func NewUserStore(path string) *UserStore {
	return &UserStore{
		users:    make(map[string]User),
		sessions: make(map[string]Session),
		path:     path,
	}
}

func (s *UserStore) Create(username, role string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, u := range s.users {
		if u.Username == username {
			return User{}, fmt.Errorf("username already exists: %s", username)
		}
	}

	id := generateID()
	token := generateToken()

	user := User{
		ID:        id,
		Username:  username,
		Token:     token,
		Role:      role,
		CreatedAt: time.Now().UTC(),
	}

	s.users[id] = user
	return user, nil
}

func (s *UserStore) SetPassword(id, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[id]
	if !ok {
		return fmt.Errorf("user not found: %s", id)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.PasswordHash = string(hash)
	s.users[id] = u
	return nil
}

func (s *UserStore) ValidatePassword(username, password string) (User, bool) {
	s.mu.RLock()
	var match *User
	for _, u := range s.users {
		if u.Username == username && u.PasswordHash != "" {
			cp := u
			match = &cp
			break
		}
	}
	s.mu.RUnlock()

	if match == nil {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return User{}, false
	}
	if bcrypt.CompareHashAndPassword([]byte(match.PasswordHash), []byte(password)) != nil {
		return User{}, false
	}
	return *match, true
}

func (s *UserStore) HasAdminWithPassword() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		if u.Role == "admin" && u.PasswordHash != "" {
			return true
		}
	}
	return false
}

func (s *UserStore) EnsureAdmin(username, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	for id, u := range s.users {
		if u.Role == "admin" && u.PasswordHash == "" {
			u.PasswordHash = string(hash)
			s.users[id] = u
			return nil
		}
	}

	id := generateID()
	token := generateToken()
	s.users[id] = User{
		ID:           id,
		Username:     username,
		Token:        token,
		Role:         "admin",
		PasswordHash: string(hash),
		CreatedAt:    time.Now().UTC(),
	}
	return nil
}

func (s *UserStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[id]; !exists {
		return fmt.Errorf("user not found: %s", id)
	}
	delete(s.users, id)
	return nil
}

func (s *UserStore) List() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]User, 0, len(s.users))
	for _, u := range s.users {
		result = append(result, u)
	}
	return result
}

func (s *UserStore) ValidateToken(token string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		if u.Token == token {
			return u, true
		}
	}
	return User{}, false
}

func (s *UserStore) CreateSession(userID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	token := generateToken()
	s.sessions[token] = Session{
		UserID:  userID,
		Expires: time.Now().Add(24 * time.Hour),
	}
	return token
}

func (s *UserStore) ValidateSession(token string) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[token]
	if !ok {
		return User{}, false
	}
	if time.Now().After(sess.Expires) {
		delete(s.sessions, token)
		return User{}, false
	}

	u, ok := s.users[sess.UserID]
	return u, ok
}

func (s *UserStore) DeleteSession(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func (s *UserStore) hasUsers() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users) > 0
}

func (s *UserStore) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie("epg_session"); err == nil {
			if _, ok := s.ValidateSession(cookie.Value); ok {
				next.ServeHTTP(w, r)
				return
			}
		}

		if !s.hasUsers() {
			next.ServeHTTP(w, r)
			return
		}

		token := r.Header.Get("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		if token == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		if _, ok := s.ValidateToken(token); !ok {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *UserStore) AdminAPIMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("epg_session")
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		if _, ok := s.ValidateSession(cookie.Value); !ok {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *UserStore) AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("epg_session")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if _, ok := s.ValidateSession(cookie.Value); !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *UserStore) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.path == "" {
		return nil
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".users-*.gob.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if err := gob.NewEncoder(tmp).Encode(userSnapshot{Users: s.users}); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	tmp.Close()

	return os.Rename(tmpPath, s.path)
}

func (s *UserStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.path == "" {
		return nil
	}

	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	var snap userSnapshot
	if err := gob.NewDecoder(f).Decode(&snap); err != nil {
		return err
	}

	s.users = snap.Users
	return nil
}

func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return hex.EncodeToString(b)
}

func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}
