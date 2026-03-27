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
)

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Token     string    `json:"token"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type UserStore struct {
	mu    sync.RWMutex
	users map[string]User
	path  string
}

type userSnapshot struct {
	Users map[string]User
}

func NewUserStore(path string) *UserStore {
	return &UserStore{
		users: make(map[string]User),
		path:  path,
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

func (s *UserStore) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(s.users) == 0 {
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
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}
