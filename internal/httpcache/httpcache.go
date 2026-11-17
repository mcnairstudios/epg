// Package httpcache wraps an HTTP client so identical requests made by many
// channels in one grab are downloaded once. It is meant for plugins whose
// request does not depend on the channel, so each channel re-fetches the
// same document. Responses are keyed by method, URL, headers and body.
package httpcache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/gavinmcnair/epg/internal/fetchcache"
)

const (
	okTTL       = 10 * time.Minute
	notFoundTTL = 30 * time.Minute
	maxBytes    = 256 << 20
)

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type response struct {
	status     int
	statusText string
	header     http.Header
	body       []byte
}

var shared = newCache(maxBytes)

func newCache(budget int64) *fetchcache.Cache[*response] {
	return fetchcache.New(fetchcache.Options[*response]{
		TTL: func(r *response, err error) time.Duration {
			if err != nil || r == nil {
				return 0
			}
			switch r.status {
			case http.StatusOK:
				return okTTL
			case http.StatusNotFound, http.StatusGone:
				return notFoundTTL
			}
			return 0
		},
		Size:     func(r *response) int64 { return int64(len(r.body)) },
		MaxBytes: budget,
	})
}

type Option func(*client)

// WithPOST also caches POST requests, keyed by their body. Use it only for
// read-only query endpoints.
func WithPOST() Option { return func(c *client) { c.post = true } }

type client struct {
	next  Doer
	cache *fetchcache.Cache[*response]
	post  bool
}

// Wrap returns a Doer that serves repeat requests from a shared cache.
// Only 200 and 404/410 responses are retained; other outcomes are shared
// with callers already waiting and then dropped, so retries still happen.
func Wrap(next Doer, opts ...Option) Doer {
	c := &client{next: next, cache: shared}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *client) Do(req *http.Request) (*http.Response, error) {
	if !c.cacheable(req) {
		return c.next.Do(req)
	}

	var body []byte
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, err
		}
		body = b
	}

	r, err := c.cache.Get(key(req, body), func() (*response, error) {
		out := req.Clone(req.Context())
		if body != nil {
			out.Body = io.NopCloser(bytes.NewReader(body))
			out.ContentLength = int64(len(body))
		}
		resp, err := c.next.Do(out)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return &response{status: resp.StatusCode, statusText: resp.Status, header: resp.Header.Clone(), body: data}, nil
	})
	if err != nil {
		return nil, err
	}

	return &http.Response{
		Status:        r.statusText,
		StatusCode:    r.status,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        r.header.Clone(),
		Body:          io.NopCloser(bytes.NewReader(r.body)),
		ContentLength: int64(len(r.body)),
		Request:       req,
	}, nil
}

func (c *client) cacheable(req *http.Request) bool {
	switch req.Method {
	case http.MethodGet:
		return true
	case http.MethodPost:
		return c.post
	}
	return false
}

func key(req *http.Request, body []byte) string {
	h := sha256.New()
	h.Write([]byte(req.Method))
	h.Write([]byte{0})
	h.Write([]byte(req.URL.String()))
	h.Write([]byte{0})
	names := make([]string, 0, len(req.Header))
	for n := range req.Header {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		h.Write([]byte(n))
		h.Write([]byte{':'})
		for _, v := range req.Header[n] {
			h.Write([]byte(v))
			h.Write([]byte{0})
		}
	}
	h.Write([]byte{1})
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}
