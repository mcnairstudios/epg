package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type tmdbCacheEntry struct {
	data      any
	expiresAt time.Time
}

var tmdbCache sync.Map
var tmdbClient = &http.Client{Timeout: 10 * time.Second}

func (s *Server) tmdbSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		jsonError(w, "query required", http.StatusBadRequest)
		return
	}

	apiKey := s.settings.Get("tmdb_api_key")
	if apiKey == "" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results":[]}`))
		return
	}

	cacheKey := "search:" + query
	if cached, ok := tmdbCache.Load(cacheKey); ok {
		entry := cached.(*tmdbCacheEntry)
		if time.Now().Before(entry.expiresAt) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(entry.data)
			return
		}
	}

	searchURL := fmt.Sprintf("https://api.themoviedb.org/3/search/multi?api_key=%s&query=%s&language=en-GB",
		url.QueryEscape(apiKey), url.QueryEscape(query))

	resp, err := tmdbClient.Get(searchURL)
	if err != nil {
		jsonError(w, "tmdb request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if results, ok := result["results"].([]any); ok && len(results) > 0 {
		tmdbCache.Store(cacheKey, &tmdbCacheEntry{data: result, expiresAt: time.Now().Add(24 * time.Hour)})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) tmdbDetails(w http.ResponseWriter, r *http.Request) {
	mediaType := r.URL.Query().Get("type")
	id := r.URL.Query().Get("id")
	if mediaType == "" || id == "" {
		jsonError(w, "type and id required", http.StatusBadRequest)
		return
	}

	apiKey := s.settings.Get("tmdb_api_key")
	if apiKey == "" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
		return
	}

	cacheKey := "detail:" + mediaType + ":" + id
	if cached, ok := tmdbCache.Load(cacheKey); ok {
		entry := cached.(*tmdbCacheEntry)
		if time.Now().Before(entry.expiresAt) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(entry.data)
			return
		}
	}

	detailURL := fmt.Sprintf("https://api.themoviedb.org/3/%s/%s?api_key=%s&language=en-GB&append_to_response=images,credits",
		url.PathEscape(mediaType), url.PathEscape(id), url.QueryEscape(apiKey))

	resp, err := tmdbClient.Get(detailURL)
	if err != nil {
		jsonError(w, "tmdb request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var result any
	json.NewDecoder(resp.Body).Decode(&result)

	tmdbCache.Store(cacheKey, &tmdbCacheEntry{data: result, expiresAt: time.Now().Add(24 * time.Hour)})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
