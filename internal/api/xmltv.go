package api

import (
	"compress/gzip"
	"fmt"
	"hash/fnv"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/gavinmcnair/epg/internal/domain"
	"github.com/gavinmcnair/epg/internal/store"
	"github.com/gavinmcnair/epg/internal/xmltv"
)

func logoKey(xmltvID string) string {
	if i := strings.Index(xmltvID, "@"); i != -1 {
		return xmltvID[:i]
	}
	return xmltvID
}

// isValidXmltvID returns true for well-formed iptv-org channel IDs like
// "BBC1.uk" or "AlJazeera.qa@English". Rejects garbage site IDs like
// "64257#22624" or "syn-carstv/32277".
func isValidXmltvID(id string) bool {
	if len(id) < 3 {
		return false
	}
	if id[0] < 'A' || (id[0] > 'Z' && id[0] < 'a') || id[0] > 'z' {
		return false
	}
	if strings.ContainsAny(id, " #/,()[]") {
		return false
	}
	base := id
	if i := strings.Index(id, "@"); i >= 0 {
		base = id[:i]
	}
	dot := strings.LastIndex(base, ".")
	if dot < 0 {
		return false
	}
	tld := base[dot+1:]
	return len(tld) >= 2 && tld == strings.ToLower(tld) && !strings.ContainsAny(tld, "0123456789")
}

// logoScore returns a sortable quality score for a channel logo URL.
// Higher is better.
func logoScore(logo string) int {
	if logo == "" || strings.Contains(logo, "placeholder") {
		return 0
	}
	return 1
}

func (s *Server) getXMLTV(w http.ResponseWriter, r *http.Request) {
	f := store.Filter{
		Source:  r.URL.Query().Get("source"),
		Channel: r.URL.Query().Get("channel"),
		Genre:   r.URL.Query().Get("genre"),
		Quality: r.URL.Query().Get("quality"),
	}

	s.writeXMLTV(w, r, f)
}

func (s *Server) getSourceXMLTV(w http.ResponseWriter, r *http.Request) {
	source := chi.URLParam(r, "*")
	source = strings.TrimSuffix(source, ".xml")
	source = strings.TrimPrefix(source, "/")
	f := store.Filter{
		Source:  source,
		Channel: r.URL.Query().Get("channel"),
		Genre:   r.URL.Query().Get("genre"),
		Quality: r.URL.Query().Get("quality"),
	}

	s.writeXMLTV(w, r, f)
}

func (s *Server) writeXMLTV(w http.ResponseWriter, r *http.Request, f store.Filter) {
	disabled := s.disabled.List()

	scheme := "https"
	if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") == "" {
		scheme = "http"
	} else if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	base := scheme + "://" + r.Host

	allChannels := s.store.GetChannels(f)

	// Deduplicate by xmltv_id: keep the highest-quality entry per id.
	// Skip channels with no valid xmltv_id (garbage site_ids, empty, etc.).
	type scored struct {
		ch    domain.Channel
		score int
	}
	best := make(map[string]scored, len(allChannels))
	for _, ch := range allChannels {
		if disabled[ch.Source] || !isValidXmltvID(ch.XmltvID) {
			continue
		}
		score := logoScore(ch.Logo)
		if existing, ok := best[ch.XmltvID]; !ok || score > existing.score {
			best[ch.XmltvID] = scored{ch: ch, score: score}
		}
	}

	channels := make([]domain.Channel, 0, len(best))
	for _, s := range best {
		ch := s.ch
		if ch.Logo != "" {
			ch.Logo = base + "/logos/" + logoKey(ch.XmltvID)
		} else if ch.IsRadio() {
			ch.Logo = base + "/logos/placeholder-radio"
		} else {
			ch.Logo = base + "/logos/placeholder-tv"
		}
		channels = append(channels, ch)
	}
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].XmltvID < channels[j].XmltvID
	})

	// Build set of valid xmltv_ids so we can filter programs.
	validIDs := make(map[string]bool, len(best))
	for id := range best {
		validIDs[id] = true
	}

	programFilter := store.Filter{
		Source:  f.Source,
		Channel: f.Channel,
		Genre:   f.Genre,
		Date:    f.Date,
	}

	allPrograms := s.store.GetPrograms(programFilter)

	// Filter programs: only valid xmltv_id references, no disabled sources.
	// Deduplicate by (channel, start) — when multiple sources cover the same
	// channel, keep the first program seen for each timeslot.
	type progKey struct {
		channel string
		start   string
	}
	seen := make(map[progKey]bool)
	programs := make([]domain.Program, 0, len(allPrograms))
	for _, p := range allPrograms {
		if disabled[p.Source] || !validIDs[p.Channel] {
			continue
		}
		k := progKey{p.Channel, p.Start.String()}
		if !seen[k] {
			seen[k] = true
			programs = append(programs, p)
		}
	}

	data, err := xmltv.Generate(channels, programs)
	if err != nil {
		http.Error(w, "failed to generate XMLTV", http.StatusInternalServerError)
		return
	}

	h := fnv.New64a()
	h.Write(data)
	etag := fmt.Sprintf(`"%x"`, h.Sum64())

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gz.Write(data)
		return
	}

	w.Write(data)
}
