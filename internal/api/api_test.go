package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gavinmcnair/epg/internal/api"
	"github.com/gavinmcnair/epg/internal/auth"
	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/grabber"
	"github.com/gavinmcnair/epg/internal/logocache"
	"github.com/gavinmcnair/epg/internal/plugin"
	"github.com/gavinmcnair/epg/internal/source"
	"github.com/gavinmcnair/epg/internal/store"
)

type testProvider struct{}

func (t *testProvider) Name() string        { return "test-source" }
func (t *testProvider) Description() string { return "test provider" }
func (t *testProvider) Channels() ([]domain.Channel, error) {
	return []domain.Channel{
		{ID: "ch1", Name: "Test Channel", Source: "test-source", Lang: "en", Genre: "news", XmltvID: "TestCh.test"},
	}, nil
}
func (t *testProvider) Fetch(ch domain.Channel, date time.Time) ([]domain.Program, error) {
	return []domain.Program{
		{
			Channel:     ch.XmltvID,
			Source:      "test-source",
			Title:       "Test Show",
			Description: "A test program",
			Category:    "news",
			Lang:        "en",
			Start:       date,
			Stop:        date.Add(time.Hour),
		},
	}, nil
}

func setupTestServer() (*api.Server, *store.Store, string) {
	plugin.Reset()
	plugin.Register(&testProvider{})

	s := store.New("")
	s.SetChannels([]domain.Channel{
		{ID: "ch1", Name: "Test Channel", Source: "test-source", Lang: "en", Genre: "news", XmltvID: "TestCh.test"},
	})

	now := time.Now().UTC().Truncate(24 * time.Hour)
	s.SetPrograms("ch1", []domain.Program{
		{
			Channel:  "TestCh.test",
			Source:   "test-source",
			Title:    "Test Show",
			Category: "news",
			Lang:     "en",
			Start:    now,
			Stop:     now.Add(time.Hour),
		},
	})

	g := grabber.New(s, 1, 1)
	ext := source.NewExternalSourceManager("")
	users := auth.NewUserStore("")

	admin, _ := users.Create("admin", "admin")
	sessionToken := users.CreateSession(admin.ID)

	disabled := store.NewDisabledStore("")
	srv := api.NewServer(s, g, ext, users, disabled, logocache.New(""))
	return srv, s, sessionToken
}

func adminCookie(token string) *http.Cookie {
	return &http.Cookie{Name: "epg_session", Value: token}
}

func TestListSources(t *testing.T) {
	srv, _, session := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sources", nil)
	req.AddCookie(adminCookie(session))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var sources []domain.Source
	json.NewDecoder(w.Body).Decode(&sources)

	if len(sources) == 0 {
		t.Fatal("expected at least one source")
	}
	if sources[0].Name != "test-source" {
		t.Fatalf("expected test-source, got %s", sources[0].Name)
	}
}

func TestListChannels(t *testing.T) {
	srv, _, session := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels", nil)
	req.AddCookie(adminCookie(session))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var channels []domain.Channel
	json.NewDecoder(w.Body).Decode(&channels)

	if len(channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(channels))
	}
}

func TestListChannelsFiltered(t *testing.T) {
	srv, _, session := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels?source=nonexistent", nil)
	req.AddCookie(adminCookie(session))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var channels []domain.Channel
	json.NewDecoder(w.Body).Decode(&channels)

	if len(channels) != 0 {
		t.Fatalf("expected 0 channels for nonexistent source, got %d", len(channels))
	}
}

func TestGetChannel(t *testing.T) {
	srv, _, session := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/ch1", nil)
	req.AddCookie(adminCookie(session))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var ch domain.Channel
	json.NewDecoder(w.Body).Decode(&ch)

	if ch.Name != "Test Channel" {
		t.Fatalf("expected Test Channel, got %s", ch.Name)
	}
}

func TestGetChannelNotFound(t *testing.T) {
	srv, _, session := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/nonexistent", nil)
	req.AddCookie(adminCookie(session))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestListPrograms(t *testing.T) {
	srv, _, session := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/programs", nil)
	req.AddCookie(adminCookie(session))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var programs []domain.Program
	json.NewDecoder(w.Body).Decode(&programs)

	if len(programs) == 0 {
		t.Fatal("expected at least one program")
	}
}

func TestReadAPIWithBearerToken(t *testing.T) {
	srv, _, _ := setupTestServer()

	// Token auth should work for read-only endpoints
	users := auth.NewUserStore("")
	apiUser, _ := users.Create("apiuser", "viewer")
	srvToken := api.NewServer(store.New(""), grabber.New(store.New(""), 1, 1), source.NewExternalSourceManager(""), users, store.NewDisabledStore(""), logocache.New(""))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sources", nil)
	req.Header.Set("Authorization", "Bearer "+apiUser.Token)
	w := httptest.NewRecorder()
	srvToken.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with Bearer token on read endpoint, got %d", w.Code)
	}

	_ = srv
}

func TestTriggerGrab(t *testing.T) {
	srv, _, session := setupTestServer()

	body := bytes.NewBufferString(`{"source":"test-source"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/grab", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie(session))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTriggerGrabBadSource(t *testing.T) {
	srv, _, session := setupTestServer()

	body := bytes.NewBufferString(`{"source":"nonexistent"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/grab", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie(session))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTriggerGrabNoSource(t *testing.T) {
	srv, _, session := setupTestServer()

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/grab", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie(session))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTriggerGrabUnauthorized(t *testing.T) {
	srv, _, _ := setupTestServer()

	body := bytes.NewBufferString(`{"source":"test-source"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/grab", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", w.Code)
	}
}

func TestGetXMLTV(t *testing.T) {
	srv, _, session := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/epg.xml", nil)
	req.AddCookie(adminCookie(session))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Fatal("expected non-empty XMLTV response")
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/xml" {
		t.Fatalf("expected application/xml, got %s", contentType)
	}
}

func TestGetSourceXMLTV(t *testing.T) {
	srv, _, session := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/epg/test-source.xml", nil)
	req.AddCookie(adminCookie(session))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
