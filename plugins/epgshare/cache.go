package epgshare

import (
	"time"

	"github.com/gavinmcnair/epg/internal/fetchcache"
)

func newCache() *fetchcache.Cache[index] {
	return fetchcache.New(fetchcache.Options[index]{
		TTL:        fetchcache.DocTTL[index](15 * time.Minute),
		MaxEntries: 4,
	})
}
